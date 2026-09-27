// UpdateControl.tsx — header entry point and dialog for in-app updates that come
// from the project's GitHub Releases.
import { useCallback, useEffect, useRef, useState } from "react";
import { Icon } from "../common/Icon";
import { formatSize } from "../lib/format";
import * as api from "../backend";
import { useI18n } from "../i18n/i18n";
import type { UpdateInfo, UpdateProgress, UpdateResult } from "../lib/types";

interface Props {
  /** Toast sink supplied by App. */
  onNotify: (message: string, detail?: string, variant?: "success" | "error" | "info") => void;
  /** Check for a new release on startup, from settings. */
  auto: boolean;
  /** Incremented by settings to run an interactive check now. */
  openSignal: number;
}

type Phase = "idle" | "downloading" | "ready" | "error";

export function UpdateControl({ onNotify, auto, openSignal }: Props) {
  const { t } = useI18n();
  const [info, setInfo] = useState<UpdateInfo | null>(null);
  const [checking, setChecking] = useState(false);
  const [open, setOpen] = useState(false);
  const [phase, setPhase] = useState<Phase>("idle");
  const [progress, setProgress] = useState<UpdateProgress | null>(null);
  const [file, setFile] = useState("");
  const [error, setError] = useState("");

  useEffect(() => {
    const off1 = api.onUpdateProgress((p) => {
      setProgress(p);
      if (p.phase === "download") setPhase("downloading");
    });
    const off2 = api.onUpdateFinished((r: UpdateResult) => {
      if (r.error) {
        setError(r.error);
        setPhase("error");
        setProgress(null);
        return;
      }
      setFile(r.path);
      setPhase("ready");
    });
    return () => { off1(); off2(); };
  }, []);

  const check = useCallback(
    async (interactive: boolean) => {
      setChecking(true);
      try {
        const found = await api.checkUpdate();
        setInfo(found);
        if (interactive) {
          if (found.available) {
            setError("");
            setPhase("idle");
            setOpen(true);
          } else {
            onNotify(t("update.upToDate"), t("update.upToDateDetail", { version: found.current }));
          }
        } else if (found.available) {
          onNotify(t("update.foundToast", { version: found.latest }), t("update.foundToastDetail"), "info");
        }
      } catch (e) {
        setInfo(null);
        if (interactive) onNotify(t("update.checkFailed"), String(e), "error");
      } finally {
        setChecking(false);
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [onNotify, t],
  );

  // A single silent check per session when the user enabled it: the button lights
  // up and a toast offers the release, while any failure stays quiet.
  const autoChecked = useRef(false);
  useEffect(() => {
    if (!auto || autoChecked.current) return;
    autoChecked.current = true;
    void check(false);
  }, [auto, check]);

  // "Check for updates now" from the settings page.
  useEffect(() => {
    if (openSignal > 0) void check(true);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [openSignal]);

  const download = async () => {
    if (!info?.url) return;
    setError("");
    setProgress(null);
    setPhase("downloading");
    await api.startUpdateDownload(info.url, info.digest);
  };

  const install = async () => {
    try {
      await api.installUpdate(file);
      onNotify(t("update.installerOpened"), t("update.installerOpenedDetail"), "success");
    } catch (e) {
      setError(String(e));
      setPhase("error");
    }
  };

  const showProgress = phase === "downloading" && progress;
  const percent = progress?.percent ?? 0;
  const speed = progress && progress.speed > 0 ? t("update.speed", { speed: formatSize(progress.speed) }) : "";

  return (
    <>
      {info?.available && (
        <button
          className="btn btn-ghost btn-small update-pill available"
          onClick={() => { setError(""); setPhase("idle"); setOpen(true); }}
          disabled={checking}
          title={t("update.available", { version: info.latest })}
        >
          <Icon name="download" size={15} />
          <span>{checking ? t("update.checking") : t("update.available", { version: info.latest })}</span>
        </button>
      )}

      {open && info && (
        <div
          className="modal-backdrop"
          onClick={(e) => {
            // Only a click on the dimmed area itself closes the dialog — clicks on
            // its children bubble here too and must not.
            if (e.target === e.currentTarget && phase !== "downloading") setOpen(false);
          }}
        >
          <div className="modal">
            <h2>{t("update.title")}</h2>
            <p className="modal-summary">
              {t("update.running")} <strong>{info.current}</strong> → {t("update.latest")}{" "}
              <strong>{info.latest}</strong>
              {info.bytes > 0 && <> · {formatSize(info.bytes)}</>}
            </p>

            {info.notes && <pre className="update-notes">{info.notes}</pre>}

            {showProgress && (
              <div className="update-download">
                <div className="progress-track">
                  <div className="progress-fill" style={{ width: `${percent}%` }} />
                </div>
                <div className="progress-meta">
                  <span>
                    {t("update.progress", {
                      percent,
                      done: formatSize(progress.bytes),
                      total: progress.total > 0 ? formatSize(progress.total) : t("update.sizeUnknown"),
                    })}
                  </span>
                  <span>{speed}</span>
                </div>
              </div>
            )}

            {phase === "ready" && (
              <p className="update-ready">
                {t("update.ready")}
                <span className="update-file" title={file}>{file.split(/[\\/]/).pop()}</span>
              </p>
            )}

            {phase === "error" && (
              <div className="modal-warning">
                <Icon name="triangle-alert" size={16} />
                <span>{error}</span>
              </div>
            )}

            <div className="modal-actions">
              {phase === "downloading" ? (
                <>
                  <button className="btn btn-ghost" onClick={() => void api.cancelUpdate()}>{t("common.cancel")}</button>
                  <button className="btn btn-primary" disabled>{t("update.downloading")}</button>
                </>
              ) : phase === "ready" ? (
                <>
                  <button
                    className="btn btn-ghost"
                    onClick={() => void api.reveal(file).catch((e) => onNotify(t("toast.revealFailed"), String(e), "error"))}
                  >
                    {t("update.openInFolder")}
                  </button>
                  <button className="btn btn-primary" onClick={() => void install()}>{t("update.install")}</button>
                </>
              ) : (
                <>
                  <button className="btn btn-ghost" onClick={() => setOpen(false)}>{t("update.later")}</button>
                  <button className="btn btn-primary" onClick={() => void download()}>
                    {t(phase === "error" ? "update.retry" : "update.download")}
                  </button>
                </>
              )}
            </div>
          </div>
        </div>
      )}
    </>
  );
}
