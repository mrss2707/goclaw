import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useHttp } from "@/hooks/use-ws";
import { useAuthStore } from "@/stores/use-auth-store";

interface LoginResponse {
  token: string;
  user: {
    user_id: string;
    tenant_id: string;
    role: string;
    locale: string;
    email: string;
  };
}

interface EmailLoginFormProps {
  onDone: () => void;
}

export function EmailLoginForm({ onDone }: EmailLoginFormProps) {
  const { t } = useTranslation("login");
  const http = useHttp();
  const setCredentials = useAuthStore((s) => s.setCredentials);
  const setTenant = useAuthStore((s) => s.setTenant);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const handleLogin = async () => {
    if (!email.trim()) {
      setError(t("emailLogin.errors.emailRequired"));
      return;
    }
    if (!password) {
      setError(t("emailLogin.errors.passwordRequired"));
      return;
    }

    setLoading(true);
    setError("");

    try {
      const res = await http.post<LoginResponse>("/v1/auth/login", {
        email: email.trim(),
        password,
      });
      setCredentials(res.token, res.user.user_id);
      setTenant(res.user.tenant_id, "Default", "default", false);
      onDone();
    } catch (err: any) {
      const msg = err?.message || t("emailLogin.errors.failedLogin");
      setError(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="login-email">{t("emailLogin.email")}</Label>
        <Input
          id="login-email"
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          placeholder="admin@example.com"
          className="text-base md:text-sm"
          autoComplete="email"
        />
      </div>

      <div className="space-y-2">
        <Label htmlFor="login-password">{t("emailLogin.password")}</Label>
        <Input
          id="login-password"
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder="••••••••"
          className="text-base md:text-sm"
          autoComplete="current-password"
        />
      </div>

      {error && <p className="text-sm text-destructive">{error}</p>}

      <Button className="w-full" onClick={handleLogin} disabled={loading}>
        {loading ? t("emailLogin.signingIn") : t("emailLogin.signIn")}
      </Button>
    </div>
  );
}
