// AppLoader.tsx — modern & elegant application startup progress screen.
import { useI18n } from "../i18n/i18n";
import brandLogo from "../assets/brand/logo.svg";

export function AppLoader() {
  const { t } = useI18n();

  return (
    <div className="app-loader-container fade-in">
      <div className="app-loader-card">
        <div className="app-loader-brand">
          <div className="app-loader-icon-wrap">
            <img src={brandLogo} alt="" width={36} height={36} />
          </div>
          <h2 className="app-loader-title">{t("app.title")}</h2>
          <p className="app-loader-subtitle">{t("app.loadingSubtitle")}</p>
        </div>

        <div className="app-loader-progress-track">
          <div className="app-loader-progress-bar" />
        </div>
      </div>
    </div>
  );
}
