import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { TooltipProvider } from "@/components/ui/tooltip";
import { InfoTip } from "@/pages/setup/info-tip";
import { useHttp } from "@/hooks/use-ws";
import { useAuthStore } from "@/stores/use-auth-store";
import { toast } from "@/stores/use-toast-store";

interface SetupResponse {
  token: string;
  user_id: string;
  tenant_id: string;
}

interface FirstAdminFormProps {
  onDone: () => void;
}

export function FirstAdminForm({ onDone }: FirstAdminFormProps) {
  const { t } = useTranslation("login");
  const http = useHttp();
  const setCredentials = useAuthStore((s) => s.setCredentials);
  const setTenant = useAuthStore((s) => s.setTenant);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const handleCreate = async () => {
    if (!email.trim()) {
      setError(t("firstAdmin.errors.emailRequired"));
      return;
    }
    if (!password) {
      setError(t("firstAdmin.errors.passwordRequired"));
      return;
    }

    setLoading(true);
    setError("");

    try {
      const res = await http.post<SetupResponse>("/v1/admin/setup", {
        email: email.trim(),
        password,
        display_name: displayName.trim(),
      });
      setCredentials(res.token, res.user_id);
      setTenant(res.tenant_id, "Default", "default", true);
      toast.success(t("firstAdmin.created"));
      onDone();
    } catch (err: any) {
      const msg = err?.message || t("firstAdmin.errors.failedCreate");
      setError(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-4">
      <div className="space-y-1">
        <h2 className="text-lg font-semibold">{t("firstAdmin.title")}</h2>
        <p className="text-sm text-muted-foreground">{t("firstAdmin.description")}</p>
      </div>

      <TooltipProvider>
        <div className="space-y-2">
          <Label className="inline-flex items-center gap-1.5">
            {t("firstAdmin.email")}
            <InfoTip text={t("firstAdmin.emailHint")} />
          </Label>
          <Input
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="admin@example.com"
            className="text-base md:text-sm"
            autoComplete="email"
          />
        </div>

        <div className="space-y-2">
          <Label className="inline-flex items-center gap-1.5">
            {t("firstAdmin.password")}
            <InfoTip text={t("firstAdmin.passwordHint")} />
          </Label>
          <Input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="••••••••"
            className="text-base md:text-sm"
            autoComplete="new-password"
          />
        </div>

        <div className="space-y-2">
          <Label className="inline-flex items-center gap-1.5">
            {t("firstAdmin.displayName")}
            <InfoTip text={t("firstAdmin.displayNameHint")} />
          </Label>
          <Input
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
            placeholder="Admin"
            className="text-base md:text-sm"
          />
        </div>
      </TooltipProvider>

      {error && <p className="text-sm text-destructive">{error}</p>}

      <Button className="w-full" onClick={handleCreate} disabled={loading}>
        {loading ? t("firstAdmin.creating") : t("firstAdmin.create")}
      </Button>
    </div>
  );
}
