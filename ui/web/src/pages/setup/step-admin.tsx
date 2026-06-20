import { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent } from "@/components/ui/card";
import { TooltipProvider } from "@/components/ui/tooltip";
import { InfoTip } from "@/pages/setup/info-tip";
import { useHttp } from "@/hooks/use-ws";
import { useAuthStore } from "@/stores/use-auth-store";
import { toast } from "@/stores/use-toast-store";

interface StepAdminProps {
  onComplete: () => void;
  onBack?: () => void;
}

interface SetupResponse {
  token: string;
  user_id: string;
  tenant_id: string;
}

interface SetupCheckResponse {
  setup_complete: boolean;
}

export function StepAdmin({ onComplete, onBack }: StepAdminProps) {
  const { t } = useTranslation("setup");
  const http = useHttp();
  const setCredentials = useAuthStore((s) => s.setCredentials);
  const setTenant = useAuthStore((s) => s.setTenant);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [alreadySetup, setAlreadySetup] = useState(false);
  const [checkingSetup, setCheckingSetup] = useState(true);

  useEffect(() => {
    http.get<SetupCheckResponse>("/v1/admin/setup")
      .then((res) => {
        if (res.setup_complete) {
          setAlreadySetup(true);
        }
      })
      .catch(() => {})
      .finally(() => setCheckingSetup(false));
  }, []);

  if (checkingSetup) {
    return null;
  }

  if (alreadySetup) {
    return (
      <Card className="py-0 gap-0">
        <CardContent className="space-y-4 px-6 py-5">
          <div className="space-y-1">
            <p className="text-sm text-muted-foreground">
              Admin account already exists. You can manage users from the dashboard.
            </p>
          </div>
          <div className="flex justify-end">
            <Button onClick={onComplete}>
              Continue
            </Button>
          </div>
        </CardContent>
      </Card>
    );
  }

  const handleCreate = async () => {
    if (!email.trim()) {
      setError(t("admin.errors.emailRequired"));
      return;
    }
    if (!password) {
      setError(t("admin.errors.passwordRequired"));
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
      toast.success(t("admin.created"));
      onComplete();
    } catch (err: any) {
      const msg = err?.message || t("admin.errors.failedCreate");
      setError(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card className="py-0 gap-0">
      <CardContent className="space-y-4 px-6 py-5">
        <TooltipProvider>
          <div className="space-y-1">
            <h2 className="text-lg font-semibold">{t("admin.title")}</h2>
            <p className="text-sm text-muted-foreground">{t("admin.description")}</p>
          </div>

          <div className="space-y-2">
            <Label className="inline-flex items-center gap-1.5">
              {t("admin.email")}
              <InfoTip text={t("admin.emailHint")} />
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
              {t("admin.password")}
              <InfoTip text={t("admin.passwordHint")} />
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
              {t("admin.displayName")}
              <InfoTip text={t("admin.displayNameHint")} />
            </Label>
            <Input
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              placeholder="Admin"
              className="text-base md:text-sm"
            />
          </div>

          {error && <p className="text-sm text-destructive">{error}</p>}

          <div className={`flex ${onBack ? "justify-between" : "justify-end"} gap-2`}>
            {onBack && (
              <Button variant="secondary" onClick={onBack}>
                ← {t("common.back")}
              </Button>
            )}
            <Button onClick={handleCreate} disabled={loading}>
              {loading ? t("admin.creating") : t("admin.create")}
            </Button>
          </div>
        </TooltipProvider>
      </CardContent>
    </Card>
  );
}
