// Sunburst.tsx — a DaisyDisk-style radial map of disk usage. Concentric rings
// show the path from the drive root (outermost) down to the folder in view
// (innermost); each segment is a child sized by the space it occupies. Click a
// folder segment to drill into it, a file segment to select it.
import { useState } from "react";
import { formatSize } from "../lib/format";
import { useI18n } from "../i18n/i18n";
import type { AnalyzeEntry } from "../lib/types";

/** One analyzed folder and its children, from the root down to the current one. */
export interface Ring {
  path: string;
  totalBytes: number;
  entries: AnalyzeEntry[];
  /** True when the listing was cancelled before it finished measuring. */
  partial?: boolean;
}

// A directory can hold thousands of children; drawing one path per child would
// kill the frame rate, so the smallest ones merge into a single muted segment.
const MAX_ARCS = 48;
// Golden-angle hue stepping gives distinct, evenly spread colours without a
// hard-coded palette, and stays legible on both themes.
export function paletteColor(index: number): string {
  const hue = (index * 137.508) % 360;
  return `hsl(${hue.toFixed(1)} 62% 58%)`;
}

const TWO_PI = Math.PI * 2;
const START = -Math.PI / 2;

interface Segment {
  key: string;
  entry: AnalyzeEntry | null; // null for the "other" aggregate
  a0: number;
  a1: number;
  color: string;
}

// segments lays a ring's children out across [a0, a1], merging the tail into one
// muted "other" segment once MAX_ARCS is reached.
function segments(ring: Ring, a0: number, a1: number): Segment[] {
  const total = ring.totalBytes > 0 ? ring.totalBytes : ring.entries.reduce((s, e) => s + e.size, 0);
  if (total <= 0 || a1 - a0 <= 0) return [];
  const span = a1 - a0;
  const out: Segment[] = [];
  let cursor = a0;
  ring.entries.slice(0, MAX_ARCS).forEach((e, i) => {
    const end = cursor + span * Math.min(1, Math.max(0, e.size / total));
    out.push({ key: e.path, entry: e, a0: cursor, a1: end, color: paletteColor(i) });
    cursor = end;
  });
  const tail = ring.entries.slice(MAX_ARCS);
  if (tail.length > 0) {
    const size = tail.reduce((s, e) => s + e.size, 0);
    out.push({ key: `${ring.path}::other`, entry: null, a0: cursor, a1: a0 + span, color: "var(--surface-2)" });
  }
  return out;
}

// anchorSpan returns the [start, end] a child occupies in its parent's ring.
function anchorSpan(ring: Ring, a0: number, a1: number, path: string): [number, number] | null {
  const total = ring.totalBytes > 0 ? ring.totalBytes : ring.entries.reduce((s, e) => s + e.size, 0);
  if (total <= 0) return null;
  let cursor = a0;
  for (const e of ring.entries) {
    const end = cursor + (a1 - a0) * Math.min(1, Math.max(0, e.size / total));
    if (e.path === path) return [cursor, end];
    cursor = end;
  }
  return null;
}

// ringSpans computes, for each ring, the angular slice its folder covers inside
// its parent — this is what makes the rings nest instead of overlapping.
function ringSpans(rings: Ring[]): { a0: number; a1: number }[] {
  const spans: { a0: number; a1: number }[] = [];
  let cur = { a0: START, a1: START + TWO_PI };
  for (let r = 0; r < rings.length; r++) {
    spans.push(cur);
    if (r + 1 < rings.length) {
      const next = anchorSpan(rings[r], cur.a0, cur.a1, rings[r + 1].path);
      if (next) cur = { a0: next[0], a1: next[1] };
    }
  }
  return spans;
}

// donutPath builds an SVG arc from r0 (inner) to r1 (outer) between two angles.
function donutPath(cx: number, cy: number, r0: number, r1: number, a0: number, a1: number): string {
  const full = a1 - a0 >= TWO_PI - 1e-6;
  if (full) {
    // A single full-circle child cannot be drawn as one arc (start meets end);
    // draw it as two half rings instead.
    const mid = a0 + Math.PI;
    return donutPath(cx, cy, r0, r1, a0, mid) + " " + donutPath(cx, cy, r0, r1, mid, a1);
  }
  const large = a1 - a0 > Math.PI ? 1 : 0;
  const x0 = cx + r1 * Math.cos(a0), y0 = cy + r1 * Math.sin(a0);
  const x1 = cx + r1 * Math.cos(a1), y1 = cy + r1 * Math.sin(a1);
  const x2 = cx + r0 * Math.cos(a1), y2 = cy + r0 * Math.sin(a1);
  const x3 = cx + r0 * Math.cos(a0), y3 = cy + r0 * Math.sin(a0);
  return `M${x0},${y0} A${r1},${r1} 0 ${large} 1 ${x1},${y1} L${x2},${y2} A${r0},${r0} 0 ${large} 0 ${x3},${y3} Z`;
}

interface Props {
  rings: Ring[];
  selected: Set<string>;
  onDrill: (entry: AnalyzeEntry, ringIndex: number) => void;
  onSelect: (entry: AnalyzeEntry) => void;
  onHover: (entry: AnalyzeEntry | null) => void;
  onUp: () => void;
}

const SIZE = 360;

export function Sunburst({ rings, selected, onDrill, onSelect, onHover, onUp }: Props) {
  const { t } = useI18n();
  const [hover, setHover] = useState<string | null>(null);

  if (rings.length === 0) return null;
  const spans = ringSpans(rings);
  const cx = SIZE / 2, cy = SIZE / 2;
  const centerR = SIZE * 0.17;
  const outerR = SIZE * 0.47;
  const ringW = (outerR - centerR) / rings.length;
  const current = rings[rings.length - 1];

  const radius = (r: number) => ({
    r0: centerR + (rings.length - 1 - r) * ringW,
    r1: centerR + (rings.length - r) * ringW,
  });

  const activate = (seg: Segment, ringIndex: number) => {
    const e = seg.entry;
    // The "other" aggregate has no entry to act on, and a skipped folder was never
    // measured — neither should drill or select.
    if (!e || e.skipped) return;
    if (e.isDir) onDrill(e, ringIndex);
    else onSelect(e);
  };

  return (
    <div className="sunburst-wrap">
      <svg viewBox={`0 0 ${SIZE} ${SIZE}`} className="sunburst" role="img" aria-label={t("analyze.chartLabel")}>
        <defs>
          <radialGradient id="sunburst-hub-grad" cx="50%" cy="50%" r="50%">
            <stop offset="0%" stopColor="var(--surface-2)" stopOpacity="0.9" />
            <stop offset="100%" stopColor="var(--surface)" stopOpacity="1" />
          </radialGradient>
          <filter id="sunburst-shadow" x="-10%" y="-10%" width="120%" height="120%">
            <feDropShadow dx="0" dy="2" stdDeviation="3" floodColor="#000000" floodOpacity="0.15" />
          </filter>
        </defs>

        {/* Ambient background ring */}
        <circle cx={cx} cy={cy} r={outerR + 4} className="sunburst-bg-track" />

        {rings.map((ring, r) => {
          const { r0, r1 } = radius(r);
          return segments(ring, spans[r].a0, spans[r].a1).map((seg) => {
            const e = seg.entry;
            const isSel = !!e && selected.has(e.path);
            const isHover = !!e && hover === e.path;
            const blocked = !!e && e.level === "blocked";
            return (
              <path
                key={seg.key}
                d={donutPath(cx, cy, r0, r1, seg.a0, seg.a1)}
                fill={seg.color}
                className={[
                  "sunburst-seg",
                  e && e.isDir && !e.skipped ? "drillable" : "",
                  isSel ? "selected" : "",
                  isHover ? "hover" : "",
                  blocked ? "blocked" : "",
                ].join(" ")}
                onClick={() => activate(seg, r)}
                onMouseEnter={() => {
                  setHover(e?.path ?? null);
                  onHover(e);
                }}
                onMouseLeave={() => {
                  setHover(null);
                  onHover(null);
                }}
              >
                <title>
                  {e
                    ? `${e.name} — ${formatSize(e.size)} (${Math.round((e.size / (ring.totalBytes || 1)) * 100)}%)`
                    : t("analyze.other")}
                </title>
              </path>
            );
          });
        })}

        {/* Outer decorative rim for center hub */}
        <circle
          cx={cx}
          cy={cy}
          r={centerR + 1.5}
          className={`sunburst-center-rim ${rings.length > 1 ? "clickable" : ""}`}
        />

        {/* Center: the folder in view; clicking it steps one level up. */}
        <circle
          cx={cx}
          cy={cy}
          r={centerR}
          className={`sunburst-center ${rings.length > 1 ? "clickable" : ""}`}
          onClick={rings.length > 1 ? onUp : undefined}
        >
          <title>{rings.length > 1 ? t("analyze.goUp") : current.path}</title>
        </circle>

        {rings.length > 1 && (
          <g className="sunburst-up-glyph" onClick={onUp}>
            <circle cx={cx} cy={cy - 22} r={9} className="sunburst-up-pill" />
            <path
              d={`M${cx - 4} ${cy - 20} L${cx} ${cy - 24} L${cx + 4} ${cy - 20} M${cx} ${cy - 24} L${cx} ${cy - 18}`}
              className="sunburst-up-arrow"
            />
          </g>
        )}

        <text
          x={cx}
          y={rings.length > 1 ? cy - 4 : cy - 6}
          className="sunburst-center-name"
          textAnchor="middle"
        >
          {(() => {
            const raw = current.path.split(/[\\/]/).filter(Boolean).pop() ?? current.path;
            return raw.length > 14 ? `${raw.slice(0, 12)}…` : raw;
          })()}
        </text>
        <text
          x={cx}
          y={rings.length > 1 ? cy + 13 : cy + 12}
          className="sunburst-center-size"
          textAnchor="middle"
        >
          {formatSize(current.totalBytes)}
        </text>

        {rings.length > 1 && (
          <text x={cx} y={cy + 27} className="sunburst-center-hint" textAnchor="middle">
            {t("analyze.goUp")}
          </text>
        )}
      </svg>
    </div>
  );
}
