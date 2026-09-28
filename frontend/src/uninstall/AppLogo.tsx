// AppLogo.tsx — an application's real icon, fetched only when its row scrolls
// into view (a machine can list hundreds of apps and each icon is a conversion
// on the backend). Falls back to the generic icon when there is none.
import { useEffect, useRef, useState } from "react";
import { Icon } from "../common/Icon";
import { appIcon } from "../backend";

// Session cache: reloading the list must not convert every icon again. "" = no icon.
const cache = new Map<string, string>();

interface Props {
  id: string;
  fallback: string;
  size?: number;
}

export function AppLogo({ id, fallback, size = 20 }: Props) {
  const box = useRef<HTMLSpanElement>(null);
  const [src, setSrc] = useState<string | undefined>(cache.get(id));

  useEffect(() => {
    if (cache.has(id)) {
      setSrc(cache.get(id));
      return;
    }
    const el = box.current;
    if (!el) return;
    let live = true;
    const fetchIcon = () => {
      appIcon(id)
        .then((url) => { cache.set(id, url || ""); if (live) setSrc(url || ""); })
        .catch(() => { cache.set(id, ""); });
    };
    // Without IntersectionObserver (old webview) load straight away.
    if (typeof IntersectionObserver === "undefined") {
      fetchIcon();
      return () => { live = false; };
    }
    const io = new IntersectionObserver((entries) => {
      if (entries.some((e) => e.isIntersecting)) {
        io.disconnect();
        fetchIcon();
      }
    }, { rootMargin: "120px" });
    io.observe(el);
    return () => { live = false; io.disconnect(); };
  }, [id]);

  return (
    <span ref={box} className="app-logo-box" style={{ display: "grid", placeItems: "center", width: "100%", height: "100%" }}>
      {src ? <img className="app-logo" src={src} alt="" draggable={false} /> : <Icon name={fallback} size={size} />}
    </span>
  );
}
