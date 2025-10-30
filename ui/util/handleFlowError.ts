import { AxiosError } from "axios";
import { Dispatch, SetStateAction } from "react";

export interface KratosErrorResponse {
  error?: {
    id: string;
  };
  redirect_browser_to: string;
  redirect_to: string;
}

const getRedirectToFromError = (err: KratosErrorResponse) => (
  window.location.href = err.redirect_browser_to? err.redirect_browser_to: err.redirect_to
);

// deal with errors coming from initializing a flow.
export const handleFlowError =
  <S>(
    flowType:
      | "login"
      | "registration"
      | "settings"
      | "recovery"
      | "verification",
    resetFlow: Dispatch<SetStateAction<S | undefined>>,
  ) =>
  async (err: AxiosError<KratosErrorResponse>) => {
    switch (err.response?.data.error?.id) {
      case "session_aal2_required":
        resetFlow(undefined);
        // 2FA is enabled and enforced, but user did not perform 2fa yet!
        window.location.href = getRedirectToFromError(err.response.data);
        return;
      case "session_already_available":
        // User is already signed in, let's redirect them to settings
        window.location.href = "./manage_details";
        return;
      case "session_refresh_required":
        // We need to re-authenticate to perform this action
        window.location.href = getRedirectToFromError(err.response.data);
        return;
      case "session_inactive":
        // No session found, redirect the user to sign in
        resetFlow(undefined);
        window.location.href = "./login";
        return;
      case "self_service_flow_return_to_forbidden":
        // The flow expired, let's request a new one.
        resetFlow(undefined);
        window.location.href = "./" + flowType;
        return;
      case "self_service_flow_expired":
        // The flow expired, let's request a new one.
        resetFlow(undefined);
        window.location.href = "./" + flowType;
        return;
      case "security_csrf_violation":
        // A CSRF violation occurred. Best to just refresh the flow!
        resetFlow(undefined);
        window.location.href = "./" + flowType;
        return;
      case "security_identity_mismatch":
        // The requested item was intended for someone else. Let's request a new flow...
        resetFlow(undefined);
        window.location.href = "./" + flowType;
        return;
      case "browser_location_change_required":
        // Ory Kratos asked us to point the user to this URL.
        window.location.href = getRedirectToFromError(err.response.data);
        return;
      case "regenerate_backup_codes":
        // The user logged in with a lookup secret and is running out of backup codes, redirect to generate a new set
        window.location.href = getRedirectToFromError(err.response.data);
        return;
    }

    switch (err.response?.status) {
      case 410:
        // The flow expired, let's request a new one.
        resetFlow(undefined);
        window.location.href = "./" + flowType;
        return;
    }

    // We are not able to handle the error? Return it.
    return Promise.reject(err);
  };
