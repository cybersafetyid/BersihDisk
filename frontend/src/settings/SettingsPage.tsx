// SettingsPage.tsx — the settings screen: a section list plus one pane each.
import { useCallback, useEffect, useState } from "react";
import { Icon } from "../common/Icon";
import { formatSize } from "../lib/format";
import { useI18n } from "../i18n/i18n";
import { useSettings, updateSettings } from "../lib/settings";
import type { AccentKey, IconKey } from "../lib/settings";
import { iconAssets, iconKeys, iconBase64 } from "../lib/appIcons";
import { APP_AUTHOR, APP_TECH, DONATION_URL, POLICY_DATE, SUPPORT_EMAIL } from "../lib/links";
import { sections, visibleSections } from "./sections";
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
  const [copiedMail, setCopiedMail] = useState(false);

  const copyMail = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(SUPPORT_EMAIL);
      setCopiedMail(true);
      onNotify(t("settings.feedbackCopied"), SUPPORT_EMAIL, "success");
      setTimeout(() => setCopiedMail(false), 2000);
    } catch (e) {
      onNotify(t("settings.feedbackCopy"), String(e), "error");
    }
  }, [onNotify, t]);

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
  const currentMeta = sections.find((s) => s.key === section);

  return (
    <section className="settings-page fade-in">
      <header className="settings-header">
        <button className="btn btn-ghost btn-small settings-back-btn" onClick={onClose}>
          <Icon name="arrow-left" size={15} />
          <span>{t("common.back")}</span>
        </button>
        <div className="settings-header-text">
          <h2 className="section-title">{t("settings.title")}</h2>
          <p className="settings-subtitle">{t("settings.subtitle")}</p>
        </div>
      </header>

      <div className="settings-layout">
        <nav className="settings-nav">
          {(["preferences", "information", "legal"] as const).map((grp) => {
            const groupItems = items.filter((s) => s.group === grp);
            if (groupItems.length === 0) return null;
            return (
              <div key={grp} className="settings-nav-group">
                <div className="settings-nav-group-label">{t(`settings.groups.${grp}`)}</div>
                <div className="settings-nav-items">
                  {groupItems.map((s) => {
                    const isActive = section === s.key;
                    return (
                      <button
                        key={s.key}
                        className={`settings-nav-item ${isActive ? "active" : ""}`}
                        onClick={() => onSection(s.key)}
                      >
                        <span className="settings-nav-icon-wrap">
                          <Icon name={s.icon} size={15} />
                        </span>
                        <span className="settings-nav-item-label">{t(`settings.sections.${s.key}`)}</span>
                      </button>
                    );
                  })}
                </div>
              </div>
            );
          })}
        </nav>

        <div className="settings-body">
          <div className="settings-body-header">
            <span className="settings-section-badge">{t(`settings.groups.${currentMeta?.group ?? "preferences"}`)}</span>
            <h3 className="settings-section-title">{t(`settings.sections.${section}`)}</h3>
          </div>

          {section === "language" && (
            <>
              <p className="settings-lead">{t("settings.languageBody")}</p>
              <div className="option-grid">
                {(["id", "en"] as const).map((l) => {
                  const isActive = settings.lang === l;
                  return (
                    <button
                      key={l}
                      className={`option-card ${isActive ? "active" : ""}`}
                      onClick={() => updateSettings({ lang: l })}
                    >
                      <span className="option-badge">{l.toUpperCase()}</span>
                      <span className="option-title">{t(`lang.${l}`)}</span>
                    </button>
                  );
                })}
              </div>
            </>
          )}

          {section === "appearance" && (
            <>
              <p className="settings-lead">{t("settings.appearanceBody")}</p>
              <div className="option-grid">
                {themes.map((o) => {
                  const isActive = settings.theme === o.value;
                  return (
                    <button
                      key={o.value}
                      className={`option-card ${isActive ? "active" : ""}`}
                      onClick={() => updateSettings({ theme: o.value })}
                    >
                      <span className="option-icon"><Icon name={o.icon} size={18} /></span>
                      <span className="option-title">{t(`theme.${o.value}`)}</span>
                    </button>
                  );
                })}
              </div>
              <p className="settings-sub">{t("settings.accentLabel")}</p>
              <div className="accent-row">
                {accents.map((a) => {
                  const isActive = settings.accent === a;
                  return (
                    <button
                      key={a}
                      className={`accent-swatch ${isActive ? "active" : ""}`}
                      data-accent={a}
                      title={t(`settings.accents.${a}`)}
                      onClick={() => updateSettings({ accent: a })}
                    >
                      <span className="accent-dot" />
                      <span className="accent-name">{t(`settings.accents.${a}`)}</span>
                    </button>
                  );
                })}
              </div>
            </>
          )}

          {section === "icon" && (
            <>
              <p className="settings-lead">{t("settings.iconBody")}</p>
              {iconSupported ? (
                <div className="option-grid">
                  {iconKeys.map((k) => {
                    const isActive = settings.icon === k;
                    return (
                      <button
                        key={k}
                        className={`option-card icon-card ${isActive ? "active" : ""}`}
                        disabled={busyIcon !== null}
                        onClick={() => void chooseIcon(k)}
                      >
                        <img src={iconAssets[k]} alt="" width={56} height={56} />
                        <span className="option-title">{t(`settings.iconOptions.${k}`)}</span>
                      </button>
                    );
                  })}
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
              <button className="btn btn-primary" onClick={onCheckUpdate}>
                {t("settings.checkNow")}
              </button>
            </>
          )}

          {section === "cache" && (
            <>
              <p className="settings-lead">{t("settings.cacheBody")}</p>
              <button className="btn btn-ghost" onClick={() => void doClearCache()} disabled={clearing}>
                {clearing ? "..." : t("settings.clear")}
              </button>
            </>
          )}

          {section === "about" && (
            <>
              <p className="settings-lead">{t("about.body")}</p>
              <div className="info-table-card">
                <Row label={t("appinfo.name")} value={t("app.title")} />
                <Row label={t("settings.versionLabel")} value={version || "—"} />
                <Row label={t("settings.authorLabel")} value={APP_AUTHOR} />
                <Row label={t("settings.techLabel")} value={APP_TECH} />
                <Row label={t("settings.licenseLabel")} value={t("settings.licenseValue")} />
              </div>
            </>
          )}

          {section === "system" && (
            machine ? (
              <>
                <div className="info-table-card">
                  <Row label={t("sysinfo.os")} value={machine.os} />
                  <Row label={t("sysinfo.arch")} value={machine.arch} />
                  <Row label={t("sysinfo.cpus")} value={String(machine.cpuCores)} />
                  <Row label={t("sysinfo.memory")} value={machine.memoryBytes > 0 ? formatSize(machine.memoryBytes) : "—"} />
                  <Row label={t("sysinfo.hostname")} value={machine.hostname || "—"} />
                  <Row label={t("sysinfo.user")} value={machine.user || "—"} />
                  <Row label={t("sysinfo.home")} value={machine.home} />
                  <Row label={t("sysinfo.goVersion")} value={machine.goVersion} />
                </div>
              </>
            ) : <p className="settings-lead">{t("sysinfo.loading")}</p>
          )}

          {section === "app" && (
            install ? (
              <>
                <div className="info-table-card">
                  <Row label={t("appinfo.name")} value={t("app.title")} />
                  <Row label={t("appinfo.version")} value={install.version} />
                  <Row label={t("appinfo.size")} value={install.bytes > 0 ? formatSize(install.bytes) : "—"} />
                  <Row label={t("appinfo.bundle")} value={install.path} />
                  <Row label={t("appinfo.dataDir")} value="localStorage" />
                </div>
              </>
            ) : <p className="settings-lead">{t("appinfo.loading")}</p>
          )}

          {section === "privacy" && (
            <div className="legal-section">
              <div className="settings-date-text">
                {t("privacy.updated")}: {POLICY_DATE}
              </div>
              <PrivacyBody />
            </div>
          )}

          {section === "terms" && (
            <div className="legal-section">
              <div className="settings-date-text">
                {t("terms.updated")}: {POLICY_DATE}
              </div>
              <TermsBody />
            </div>
          )}

          {section === "feedback" && (
            <div className="feedback-section">
              <p className="settings-lead">{t("settings.feedbackBody")}</p>
              <div className="feedback-card">
                <div className="feedback-card-info">
                  <span className="feedback-card-label">{t("settings.sections.feedback")}</span>
                  <span className="feedback-card-email">{SUPPORT_EMAIL}</span>
                </div>

                <div className="feedback-meta-preview">
                  <span className="feedback-meta-title">{t("sysinfo.title")}:</span>
                  <span className="feedback-tag">BersihDisk v{version || "1.0"}</span>
                  {machine?.os && <span className="feedback-tag">{machine.os}</span>}
                  {machine?.arch && <span className="feedback-tag">{machine.arch}</span>}
                </div>

                <div className="feedback-card-actions">
                  <button
                    className="btn btn-primary"
                    onClick={() => openLink(
                      `mailto:${SUPPORT_EMAIL}?subject=${encodeURIComponent(`BersihDisk ${version} — ${machine?.os ?? ""}/${machine?.arch ?? ""}`)}`,
                    )}
                  >
                    {t("settings.feedbackOpen")}
                  </button>
                  <button
                    className="btn btn-ghost"
                    onClick={() => void copyMail()}
                  >
                    {copiedMail ? t("common.copied") : t("settings.feedbackCopy")}
                  </button>
                </div>
              </div>
            </div>
          )}

          {section === "donate" && (
            <>
              <p className="settings-lead">{t("settings.donateBody")}</p>
              {DONATION_URL ? (
                <button className="btn btn-primary" onClick={() => openLink(DONATION_URL)}>
                  {t("settings.donateOpen")}
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
  const paragraphs = tl("privacy.body");
  return (
    <div className="legal-article">
      {paragraphs.map((p, i) => (
        <div key={i} className="legal-item">
          <span className="legal-num">{i + 1}.</span>
          <p className="legal-text">{p}</p>
        </div>
      ))}
    </div>
  );
}

function TermsBody() {
  const { tl } = useI18n();
  const paragraphs = tl("terms.body");
  return (
    <div className="legal-article">
      {paragraphs.map((p, i) => (
        <div key={i} className="legal-item">
          <span className="legal-num">{i + 1}.</span>
          <p className="legal-text">{p}</p>
        </div>
      ))}
    </div>
  );
}
