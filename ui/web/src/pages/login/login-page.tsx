import { useState, useEffect } from "react";
import { useNavigate, useLocation } from "react-router";
import { useTranslation } from "react-i18next";
import { useHttp } from "@/hooks/use-ws";
import { ROUTES } from "@/lib/constants";
import { LoginLayout } from "./login-layout";
import { FirstAdminForm } from "./first-admin-form";
import { EmailLoginForm } from "./email-login-form";

interface SetupCheckResponse {
  setup_complete: boolean;
}

export function LoginPage() {
  const { t } = useTranslation("login");
  const http = useHttp();
  const navigate = useNavigate();
  const location = useLocation();

  const [checking, setChecking] = useState(true);
  const [setupComplete, setSetupComplete] = useState(false);
  const [checkError, setCheckError] = useState(false);

  const from =
    (location.state as { from?: { pathname: string } })?.from?.pathname ??
    ROUTES.OVERVIEW;

  useEffect(() => {
    http.get<SetupCheckResponse>("/v1/admin/setup")
      .then((res) => {
        setSetupComplete(res.setup_complete);
      })
      .catch(() => {
        setCheckError(true);
      })
      .finally(() => setChecking(false));
  }, []);

  function handleDone() {
    navigate(from, { replace: true });
  }

  if (checking) {
    return (
      <LoginLayout subtitle={t("subtitle")}>
        <div className="flex justify-center py-4">
          <div className="h-6 w-6 animate-spin rounded-full border-2 border-primary border-t-transparent" />
        </div>
      </LoginLayout>
    );
  }

  if (checkError) {
    return (
      <LoginLayout subtitle={t("subtitle")}>
        <p className="text-center text-sm text-destructive">
          {t("cannotCheckSetup")}
        </p>
      </LoginLayout>
    );
  }

  return (
    <LoginLayout subtitle={t("subtitle")}>
      {setupComplete ? (
        <EmailLoginForm onDone={handleDone} />
      ) : (
        <FirstAdminForm onDone={handleDone} />
      )}
    </LoginLayout>
  );
}
