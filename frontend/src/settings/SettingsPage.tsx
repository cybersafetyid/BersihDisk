// SettingsPage.tsx — the settings screen: a section list plus one pane each.
import { useCallback, useEffect, useState } from "react";
import { Icon } from "../common/Icon";
import { formatSize } from "../lib/format";
import { useI18n } from "../i18n/i18n";
import { useSettings, updateSettings } from "../lib/settings";
import type { AccentKey, IconKey } from "../lib/settings";
import { iconAssets, iconKeys, iconBase64 } from "../lib/appIcons";
import { APP_AUTHOR, APP_TECH, DONATION_URL, POLICY_DATE, SUPPORT_EMAIL } from "../lib/links";
import { visibleSections } from "./sections";
import type { Section } from "./sections";
import * as api from "../backend";
import type { AppInstallInfo, MachineInfo } from "../lib/types";
import type { Theme } from "../lib/types";

interface Props {
  section: Section;
  onSection: (s: Section) => void;
  onClose: () => void;
  onNotify: (message: string, detail?: string, variant?: "success" | "error" | "info") => void;
  iconSupported: boolean;
  onCheckUpdate: () => void;
}

const themes: { value: Theme; icon: string }[] = [
  { value: "light", icon: "sun" },
  { value: "dark", icon: "moon" },
  { value: "system", icon: "laptop" },
];

const accents: AccentKey[] = ["blue", "violet", "teal", "amber", "rose"];

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="info-row">
      <span className="info-label">{label}</span>
      <span className="info-value" title={value}>{value}</span>
    </div>
  );
}

export function SettingsPage({ section, onSection, onClose, onNotify, iconSupported, onCheckUpdate }: Props) {
  const { t } = useI18n();
  const settings = useSettings();
  const [machine, setMachine] = useState<MachineInfo | null>(null);
  const [install, setInstall] = useState<AppInstallInfo | null>(null);
  const [version, setVersion] = useState("");
  const [busyIcon, setBusyIcon] = useState<IconKey | null>(null);
  const [clearing, setClearing] = useState(false);

  // Read once per visit: the About and Feedback panes need the version and the
  // platform in the mail subject, not only when the info sections are open.
  useEffect(() => {
    api.appInfo()
      .then((info) => setVersion(String(info.version ?? "")))
      .catch(() => setVersion(""));
    api.systemInfo()
      .then(setMachine)
      .catch((e) => onNotify(t("sysinfo.failed"), String(e), "error"));
  }, [onNotify, t]);

  useEffect(() => {
    if (section !== "app" || install) return;
    api.appDetails().then(setInstall).catch((e) => onNotify(t("appinfo.failed"), String(e), "error"));
  }, [section, install, onNotify, t]);

  const chooseIcon = useCallback(async (key: IconKey) => {
    setBusyIcon(key);
    try {
      await api.setAppIcon(await iconBase64(key));
      updateSettings({ icon: key });
    } catch (e) {
      onNotify(t("settings.iconBody"), String(e), "error");
    } finally {
      setBusyIcon(null);
    }
  }, [onNotify, t]);

  const doClearCache = async () => {
    setClearing(true);
    try {
      const res = await api.clearCache();
      if (res.files === 0) onNotify(t("settings.cacheAlreadyEmpty"));
      else onNotify(t("settings.cacheCleared"), t("settings.cacheClearedDetail", {
        size: formatSize(res.bytes), count: res.files,
      }), "success");
    } catch (e) {
      onNotify(t("settings.cacheFailed"), String(e), "error");
    } finally {
      setClearing(false);
    }
  };

  const openLink = (url: string) => {
    api.openLink(url).catch((e) => onNotify(t("toast.revealFailed"), String(e), "error"));
  };

  const items = visibleSections(iconSupported);

  return (
    <section className="settings-page fade-in">
      <header className="settings-header">
        <button className="btn btn-ghost" onClick={onClose}>{t("common.back")}</button>
        <div>
          <h2 className="section-title">{t("settings.title")}</h2>
          <p className="settings-subtitle">{t("settings.subtitle")}</p>
        </div>
      </header>

      <div className="settings-layout">
        <nav className="settings-nav">
          {items.map((s) => (
            <button
              key={s.key}
              className={`settings-nav-item ${section === s.key ? "active" : ""}`}
              onClick={() => onSection(s.key)}
            >
              <Icon name={s.icon} size={15} />
              <span>{t(`settings.sections.${s.key}`)}</span>
            </button>
          ))}
        </nav>

        <div className="settings-body">
          <h3 className="settings-section-title">{t(`settings.sections.${section}`)}</h3>

          {section === "language" && (
            <>
              <p className="settings-lead">{t("settings.languageBody")}</p>
              <div className="option-grid">
                {(["id", "en"] as const).map((l) => (
                  <button
                    key={l}
                    className={`option-card ${settings.lang === l ? "active" : ""}`}
                    onClick={() => updateSettings({ lang: l })}
                  >
                    <span className="option-title">{t(`lang.${l}`)}</span>
                    <span className="option-desc">{l.toUpperCase()}</span>
                  </button>
                ))}
              </div>
            </>
          )}

          {section === "appearance" && (
            <>
              <p className="settings-lead">{t("settings.appearanceBody")}</p>
              <div className="option-grid">
                {themes.map((o) => (
                  <button
                    key={o.value}
                    className={`option-card ${settings.theme === o.value ? "active" : ""}`}
                    onClick={() => updateSettings({ theme: o.value })}
                  >
                    <span className="option-icon"><Icon name={o.icon} size={18} /></span>
                    <span className="option-title">{t(`theme.${o.value}`)}</span>
                  </button>
                ))}
              </div>
              <p className="settings-sub">{t("settings.accentLabel")}</p>
              <div className="accent-row">
                {accents.map((a) => (
                  <button
                    key={a}
                    className={`accent-swatch ${settings.accent === a ? "active" : ""}`}
                    data-accent={a}
                    title={t(`settings.accents.${a}`)}
                    onClick={() => updateSettings({ accent: a })}
                  >
                    <span className="accent-dot" />
                    <span className="accent-name">{t(`settings.accents.${a}`)}</span>
                  </button>
                ))}
              </div>
            </>
          )}

          {section === "icon" && (
            <>
              <p className="settings-lead">{t("settings.iconBody")}</p>
              {iconSupported ? (
                <div className="option-grid">
                  {iconKeys.map((k) => (
                    <button
                      key={k}
                      className={`option-card icon-card ${settings.icon === k ? "active" : ""}`}
                      disabled={busyIcon !== null}
                      onClick={() => void chooseIcon(k)}
                    >
                      <img src={iconAssets[k]} alt="" width={56} height={56} />
                      <span className="option-title">{t(`settings.iconOptions.${k}`)}</span>
                    </button>
                  ))}
                </div>
              ) : (
                <p className="settings-lead">{t("settings.iconUnsupported")}</p>
              )}
            </>
          )}

          {section === "updates" && (
            <>
              <p className="settings-lead">{t("settings.updatesBody")}</p>
              <label className="switch-row">
                <input
                  type="checkbox"
                  checked={settings.autoUpdate}
                  onChange={(e) => updateSettings({ autoUpdate: e.target.checked })}
                />
                <span>
                  <span className="switch-title">{t("update.autoLabel")}</span>
                  <span className="switch-desc">{t("update.autoHint")}</span>
                </span>
              </label>
              <button className="btn btn-primary" onClick={onCheckUpdate}>{t("settings.checkNow")}</button>
            </>
          )}

          {section === "cache" && (
            <>
              <p className="settings-lead">{t("settings.cacheBody")}</p>
              <button className="btn btn-ghost" onClick={() => void doClearCache()} disabled={clearing}>
                {t("settings.clear")}
              </button>
            </>
          )}

          {section === "about" && (
            <>
              <p className="settings-lead">{t("about.body")}</p>
              <Row label={t("appinfo.name")} value={t("app.title")} />
              <Row label={t("settings.versionLabel")} value={version || "—"} />
              <Row label={t("settings.authorLabel")} value={APP_AUTHOR} />
              <Row label={t("settings.techLabel")} value={APP_TECH} />
              <Row label={t("settings.licenseLabel")} value={t("settings.licenseValue")} />
            </>
          )}

          {section === "system" && (
            machine ? (
              <>
                <Row label={t("sysinfo.os")} value={machine.os} />
                <Row label={t("sysinfo.arch")} value={machine.arch} />
                <Row label={t("sysinfo.cpus")} value={String(machine.cpuCores)} />
                <Row label={t("sysinfo.memory")} value={machine.memoryBytes > 0 ? formatSize(machine.memoryBytes) : "—"} />
                <Row label={t("sysinfo.hostname")} value={machine.hostname || "—"} />
                <Row label={t("sysinfo.user")} value={machine.user || "—"} />
                <Row label={t("sysinfo.home")} value={machine.home} />
                <Row label={t("sysinfo.goVersion")} value={machine.goVersion} />
              </>
            ) : <p className="settings-lead">{t("sysinfo.loading")}</p>
          )}

          {section === "app" && (
            install ? (
              <>
                <Row label={t("appinfo.name")} value={t("app.title")} />
                <Row label={t("appinfo.version")} value={install.version} />
                <Row label={t("appinfo.size")} value={install.bytes > 0 ? formatSize(install.bytes) : "—"} />
                <Row label={t("appinfo.bundle")} value={install.path} />
                <Row label={t("appinfo.dataDir")} value="localStorage" />
              </>
            ) : <p className="settings-lead">{t("appinfo.loading")}</p>
          )}

          {section === "privacy" && (
            <>
              <p className="settings-sub">{t("privacy.updated")}: {POLICY_DATE}</p>
              <PrivacyBody />
            </>
          )}

          {section === "terms" && (
            <>
              <p className="settings-sub">{t("terms.updated")}: {POLICY_DATE}</p>
              <TermsBody />
            </>
          )}

          {section === "feedback" && (
            <>
              <p className="settings-lead">{t("settings.feedbackBody")}</p>
              <button
                className="btn btn-primary"
                onClick={() => openLink(
                  `mailto:${SUPPORT_EMAIL}?subject=${encodeURIComponent(`BersihDisk ${version} — ${machine?.os ?? ""}/${machine?.arch ?? ""}`)}`,
                )}
              >
                <Icon name="mail" size={15} /> {t("settings.feedbackOpen")}
              </button>
              <p className="settings-sub">{SUPPORT_EMAIL}</p>
            </>
          )}

          {section === "donate" && (
            <>
              <p className="settings-lead">{t("settings.donateBody")}</p>
              {DONATION_URL ? (
                <button className="btn btn-primary" onClick={() => openLink(DONATION_URL)}>
                  <Icon name="heart" size={15} /> {t("settings.donateOpen")}
                </button>
              ) : (
                <p className="settings-sub">{t("settings.donateNotConfigured")}</p>
              )}
            </>
          )}
        </div>
      </div>
    </section>
  );
}

function PrivacyBody() {
  const { tl } = useI18n();
  return <div className="prose">{tl("privacy.body").map((p, i) => <p key={i}>{p}</p>)}</div>;
}

function TermsBody() {
  const { tl } = useI18n();
  return <div className="prose">{tl("terms.body").map((p, i) => <p key={i}>{p}</p>)}</div>;
}
