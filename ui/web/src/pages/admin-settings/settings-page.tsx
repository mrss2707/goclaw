import { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { Settings } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { PageHeader } from "@/components/shared/page-header";
import { InfoTip } from "@/pages/setup/info-tip";
import { TooltipProvider } from "@/components/ui/tooltip";
import { useHttp } from "@/hooks/use-ws";
import { toast } from "@/stores/use-toast-store";

interface SupermeoConfig {
  google: {
    client_id: string;
    client_secret: string;
    redirect_url: string;
    enabled: boolean;
  };
  smtp: {
    host: string;
    port: string;
    username: string;
    password: string;
    from_name: string;
    enabled: boolean;
  };
}

export function AdminSettingsPage() {
  const { t } = useTranslation("admin-settings");
  const http = useHttp();
  const [loading, setLoading] = useState(true);
  const [savingGoogle, setSavingGoogle] = useState(false);
  const [savingSMTP, setSavingSMTP] = useState(false);
  const [config, setConfig] = useState<SupermeoConfig | null>(null);

  const [googleForm, setGoogleForm] = useState({
    client_id: "",
    client_secret: "",
    redirect_url: "",
  });

  const [smtpForm, setSmtpForm] = useState({
    host: "",
    port: "587",
    username: "",
    password: "",
    from_name: "GoClaw",
  });

  useEffect(() => {
    http.get<SupermeoConfig>("/v1/admin/supermeo-config")
      .then((cfg) => {
        setConfig(cfg);
        if (cfg.google.client_id) {
          setGoogleForm({
            client_id: cfg.google.client_id,
            client_secret: "",
            redirect_url: cfg.google.redirect_url || "",
          });
        }
        if (cfg.smtp.host) {
          setSmtpForm({
            host: cfg.smtp.host,
            port: cfg.smtp.port || "587",
            username: cfg.smtp.username,
            password: "",
            from_name: cfg.smtp.from_name || "GoClaw",
          });
        }
      })
      .catch(() => toast.error(t("google.errors.loadFailed")))
      .finally(() => setLoading(false));
  }, []);

  const saveGoogle = async () => {
    setSavingGoogle(true);
    try {
      const updated = await http.put<SupermeoConfig>("/v1/admin/supermeo-config", {
        google: googleForm,
      });
      setConfig(updated);
      toast.success(t("google.saved"));
    } catch {
      toast.error(t("google.errors.saveFailed"));
    } finally {
      setSavingGoogle(false);
    }
  };

  const saveSMTP = async () => {
    setSavingSMTP(true);
    try {
      const updated = await http.put<SupermeoConfig>("/v1/admin/supermeo-config", {
        smtp: smtpForm,
      });
      setConfig(updated);
      toast.success(t("smtp.saved"));
    } catch {
      toast.error(t("smtp.errors.saveFailed"));
    } finally {
      setSavingSMTP(false);
    }
  };

  return (
    <div className="p-4 sm:p-6 pb-10">
      <PageHeader
        title={t("title")}
        description={<Settings className="inline h-4 w-4 mr-1" />}
      />

      <div className="mt-6 space-y-6">
        {/* Google OAuth2 Section */}
        <Card>
          <CardContent className="space-y-4 px-6 py-5">
            <TooltipProvider>
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="text-lg font-semibold">{t("google.title")}</h3>
                  <p className="text-sm text-muted-foreground">{t("google.description")}</p>
                </div>
                <Badge variant={config?.google.enabled ? "default" : "secondary"}>
                  {config?.google.enabled ? t("google.enabled") : t("google.disabled")}
                </Badge>
              </div>

              <div className="space-y-2">
                <Label className="inline-flex items-center gap-1.5">
                  {t("google.clientId")}
                  <InfoTip text={t("google.clientIdHint")} />
                </Label>
                <Input
                  value={googleForm.client_id}
                  onChange={(e) => setGoogleForm({ ...googleForm, client_id: e.target.value })}
                  placeholder="123456789-xxx.apps.googleusercontent.com"
                />
              </div>

              <div className="space-y-2">
                <Label className="inline-flex items-center gap-1.5">
                  {t("google.clientSecret")}
                  <InfoTip text={t("google.clientSecretHint")} />
                </Label>
                <Input
                  type="password"
                  value={googleForm.client_secret}
                  onChange={(e) => setGoogleForm({ ...googleForm, client_secret: e.target.value })}
                  placeholder={config?.google.enabled ? "(hidden for security)" : ""}
                />
              </div>

              <div className="space-y-2">
                <Label className="inline-flex items-center gap-1.5">
                  {t("google.redirectUrl")}
                  <InfoTip text={t("google.redirectUrlHint")} />
                </Label>
                <Input
                  value={googleForm.redirect_url}
                  onChange={(e) => setGoogleForm({ ...googleForm, redirect_url: e.target.value })}
                  placeholder="https://yourdomain.com/v1/auth/google/callback"
                />
              </div>

              <div className="flex justify-end">
                <Button onClick={saveGoogle} disabled={savingGoogle || loading}>
                  {savingGoogle ? t("google.saving") : t("google.save")}
                </Button>
              </div>
            </TooltipProvider>
          </CardContent>
        </Card>

        {/* SMTP Section */}
        <Card>
          <CardContent className="space-y-4 px-6 py-5">
            <TooltipProvider>
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="text-lg font-semibold">{t("smtp.title")}</h3>
                  <p className="text-sm text-muted-foreground">{t("smtp.description")}</p>
                </div>
                <Badge variant={config?.smtp.enabled ? "default" : "secondary"}>
                  {config?.smtp.enabled ? t("smtp.enabled") : t("smtp.disabled")}
                </Badge>
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label className="inline-flex items-center gap-1.5">
                    {t("smtp.host")}
                    <InfoTip text={t("smtp.hostHint")} />
                  </Label>
                  <Input
                    value={smtpForm.host}
                    onChange={(e) => setSmtpForm({ ...smtpForm, host: e.target.value })}
                    placeholder="smtp.gmail.com"
                  />
                </div>

                <div className="space-y-2">
                  <Label className="inline-flex items-center gap-1.5">
                    {t("smtp.port")}
                    <InfoTip text={t("smtp.portHint")} />
                  </Label>
                  <Input
                    value={smtpForm.port}
                    onChange={(e) => setSmtpForm({ ...smtpForm, port: e.target.value })}
                    placeholder="587"
                  />
                </div>
              </div>

              <div className="space-y-2">
                <Label className="inline-flex items-center gap-1.5">
                  {t("smtp.username")}
                  <InfoTip text={t("smtp.usernameHint")} />
                </Label>
                <Input
                  value={smtpForm.username}
                  onChange={(e) => setSmtpForm({ ...smtpForm, username: e.target.value })}
                  placeholder="yourname@gmail.com"
                />
              </div>

              <div className="space-y-2">
                <Label className="inline-flex items-center gap-1.5">
                  {t("smtp.password")}
                  <InfoTip text={t("smtp.passwordHint")} />
                </Label>
                <Input
                  type="password"
                  value={smtpForm.password}
                  onChange={(e) => setSmtpForm({ ...smtpForm, password: e.target.value })}
                  placeholder={config?.smtp.enabled ? "(hidden for security)" : ""}
                />
              </div>

              <div className="space-y-2">
                <Label className="inline-flex items-center gap-1.5">
                  {t("smtp.fromName")}
                  <InfoTip text={t("smtp.fromNameHint")} />
                </Label>
                <Input
                  value={smtpForm.from_name}
                  onChange={(e) => setSmtpForm({ ...smtpForm, from_name: e.target.value })}
                  placeholder="GoClaw"
                />
              </div>

              <div className="flex justify-end">
                <Button onClick={saveSMTP} disabled={savingSMTP || loading}>
                  {savingSMTP ? t("smtp.saving") : t("smtp.save")}
                </Button>
              </div>
            </TooltipProvider>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
