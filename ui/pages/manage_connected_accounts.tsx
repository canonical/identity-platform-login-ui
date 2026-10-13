import type { NextPage } from "next";
import React, { useCallback, useEffect, useMemo, useState } from "react";
import { useRouter } from "next/router";
import {
  Button,
  ConfirmationButton,
  Spinner,
} from "@canonical/react-components";
import {
  SettingsFlow,
  UiNodeInputAttributes,
  UiNodeTextAttributes,
  UpdateSettingsFlowBody,
} from "@ory/client";
import PageLayout from "../components/PageLayout";
import { kratos } from "../api/kratos";
import { handleFlowError } from "../util/handleFlowError";
import { AxiosError } from "axios";
import { getLoggedInName } from "../util/selfServeHelpers";
import { List, Icon, useToastNotification } from "@canonical/react-components";
import { getProviderImage } from "../util/logos";
import { getCsrfToken } from "../util/getCsrfNode";
import {
  getSsoLinkNodeId,
  isSsoNode,
  isSsoUnlinkBtn,
  SSO_UNLINK_FIELD,
  SSO_UNLINK_LABEL_PREFIX,
} from "../util/constants";
import { FlowMessages } from "../components/FlowMessages";

type ProviderConnectionAction = "link" | "unlink";
type ConnectionState = "none" | "allDisconnected" | "someConnected";
type SettingsFlowWithRedirect = SettingsFlow & {
  redirect_to?: string;
};

interface ProviderState {
  id: string;
  label: string;
  name: ProviderConnectionAction;
  disabled: boolean;
}

// A company sign-in attached to the account. The backend adds these to the
// settings flow: a text node describing it, and a submit to unlink it whose
// value is the connection id.
interface CompanySignInState {
  id: string;
  label: string;
  description?: string;
  disabled: boolean;
}

const CONNECTION_TEXT: Record<ConnectionState, string> = {
  none: "No accounts to connect right now. Services will appear here once they’ve been made available to connect.",
  allDisconnected:
    "You haven’t connected any accounts yet. Connect an account from a service as another way to sign in quickly and securely.",
  someConnected:
    "You’ve connected the accounts below. Connect an account from a service as another way to sign in quickly and securely.",
};

const buildOidcProviderStates = (flow?: SettingsFlow): ProviderState[] => {
  if (!flow) return [];

  const byProvider: Record<string, ProviderState> = {};

  for (const node of flow.ui?.nodes ?? []) {
    if (node.group !== "oidc") continue;

    const attributes = node.attributes as UiNodeInputAttributes & {
      name: ProviderConnectionAction;
      value: string;
      disabled?: boolean;
    };

    const providerId = String(attributes.value).toLowerCase();

    const meta = node.meta as
      | {
          label?: {
            text?: string;
            context?: { provider?: string };
          };
        }
      | undefined;

    const label = meta?.label?.context?.provider ?? providerId;

    byProvider[providerId] = {
      id: providerId,
      label,
      name: attributes.name,
      disabled: Boolean(attributes.disabled),
    };
  }

  return Object.values(byProvider);
};

const buildCompanySignInStates = (
  flow?: SettingsFlow,
): CompanySignInState[] => {
  if (!flow) return [];

  const ssoNodes = (flow.ui?.nodes ?? []).filter(isSsoNode);
  const descriptions: Record<string, string> = {};
  for (const node of ssoNodes) {
    if (node.type !== "text") continue;
    const attributes = node.attributes as UiNodeTextAttributes;
    descriptions[attributes.id] = attributes.text.text;
  }

  return ssoNodes.filter(isSsoUnlinkBtn).map((node) => {
    const attributes = node.attributes as UiNodeInputAttributes;
    const id = String(attributes.value);
    const labelText = node.meta.label?.text ?? id;
    return {
      id,
      label: labelText.startsWith(SSO_UNLINK_LABEL_PREFIX)
        ? labelText.slice(SSO_UNLINK_LABEL_PREFIX.length)
        : labelText,
      description: descriptions[getSsoLinkNodeId(id)],
      disabled: Boolean(attributes.disabled),
    };
  });
};

const ManageConnectedAccounts: NextPage = () => {
  const [flow, setFlow] = useState<SettingsFlow>();

  const router = useRouter();
  const toastNotify = useToastNotification();
  const { return_to: returnTo } = router.query;

  const userName = getLoggedInName(flow);
  const providers = useMemo(() => buildOidcProviderStates(flow), [flow]);
  const companySignIns = useMemo(() => buildCompanySignInStates(flow), [flow]);

  useEffect(() => {
    if (!router.isReady || providers.length === 0) return;

    const pendingProviderId = window.sessionStorage.getItem(
      "pending_provider_link",
    );
    if (!pendingProviderId) return;

    const pendingProvider = providers.find((p) => p.id === pendingProviderId);
    const isLinked = pendingProvider?.name === "unlink";

    if (isLinked) {
      toastNotify.success(
        `Your ${pendingProvider.label} account is now connected and can be used to sign in.`,
        undefined,
        "Account connected successfully",
      );
    }

    window.sessionStorage.removeItem("pending_provider_link");
  }, [router.isReady, providers]);

  useEffect(() => {
    if (!router.isReady || flow) {
      return;
    }

    kratos
      .createBrowserSettingsFlow({
        returnTo: returnTo ? String(returnTo) : undefined,
      })
      .then(({ data }) => {
        setFlow(data);
      })
      .catch(handleFlowError("settings", setFlow))
      .catch(async (err: AxiosError<string>) => {
        if (
          typeof err.response?.data === "string" &&
          err.response.data.trim() === "Failed to create settings flow"
        ) {
          window.location.href = `./login?return_to=${window.location.pathname}`;
          return;
        }

        return Promise.reject(err);
      });
  }, [router, router.isReady, returnTo, flow]);

  const handleSubmit = useCallback(
    (action: ProviderConnectionAction, providerId: string) => {
      const label =
        providers.find((p) => p.id === providerId)?.label ?? providerId;

      return kratos
        .updateSettingsFlow({
          flow: String(flow?.id),
          updateSettingsFlowBody: {
            method: "oidc",
            [action]: providerId,
          },
        })
        .then((result) => {
          const data = result.data as SettingsFlowWithRedirect;
          if (data.redirect_to) {
            window.sessionStorage.setItem("pending_provider_link", providerId);
            window.location.href = data.redirect_to;
            return;
          }
          setFlow(data);
          if (action === "unlink") {
            toastNotify.success(
              `Your ${label} account has been disconnected.`,
              undefined,
              "Account disconnected successfully",
            );
          }
        })
        .catch((err: AxiosError) => {
          const errorAction = action === "link" ? "connect" : "disconnect";
          toastNotify.failure(
            `Failed to ${errorAction} account`,
            undefined,
            err?.message,
          );
          handleFlowError("settings", setFlow);
        });
    },
    [flow, router],
  );

  // A refusal (e.g. the account's only sign-in method) comes back as 200 with
  // the flow's messages, so success is judged by the returned flow alone.
  const handleCompanySignInUnlink = useCallback(
    (connectionId: string) => {
      const label =
        companySignIns.find((c) => c.id === connectionId)?.label ??
        connectionId;

      return kratos
        .updateSettingsFlow({
          flow: String(flow?.id),
          updateSettingsFlowBody: {
            [SSO_UNLINK_FIELD]: connectionId,
            csrf_token: getCsrfToken(flow?.ui.nodes),
          } as unknown as UpdateSettingsFlowBody,
        })
        .then((result) => {
          const data = result.data as SettingsFlowWithRedirect;
          if (data.redirect_to) {
            window.location.href = data.redirect_to;
            return;
          }
          setFlow(data);
          const isGone = !buildCompanySignInStates(data).some(
            (c) => c.id === connectionId,
          );
          const hasError = data.ui.messages?.some((m) => m.type === "error");
          if (isGone && !hasError) {
            toastNotify.success(
              `Your ${label} sign-in has been disconnected.`,
              undefined,
              "Account disconnected successfully",
            );
          }
        })
        .catch(handleFlowError("settings", setFlow))
        .catch((err: AxiosError) => {
          toastNotify.failure(
            "Failed to disconnect account",
            undefined,
            err?.message,
          );
        });
    },
    [flow, companySignIns],
  );

  const connectionState = useMemo<ConnectionState>(() => {
    if (companySignIns.length) return "someConnected";
    if (!providers.length) return "none";
    return providers.some((p) => p.name === "unlink")
      ? "someConnected"
      : "allDisconnected";
  }, [providers, companySignIns]);

  if (!flow) {
    return <Spinner />;
  }

  return (
    <PageLayout title="Connected accounts" isSelfServe={true} user={userName}>
      {flow && (
        <div>
          <FlowMessages messages={flow.ui.messages} types={["error"]} />
          <p>{CONNECTION_TEXT[connectionState]}</p>
          {connectionState !== "none" && (
            <>
              <p className="p-heading--5">Manage accounts</p>
              <List
                items={providers.map(({ id, label, disabled, name }) => (
                  <div key={id} className="provider">
                    <img
                      src={getProviderImage(label)}
                      alt={`${label} logo`}
                      className="provider-logo"
                    />
                    <span>{label}</span>
                    {name === "link" ? (
                      <Button
                        className="link-provider-btn"
                        appearance="positive"
                        disabled={disabled}
                        onClick={() => handleSubmit("link", id)}
                        hasIcon
                      >
                        <Icon name="get-link" />
                        <span>Connect</span>
                      </Button>
                    ) : (
                      <ConfirmationButton
                        disabled={disabled}
                        appearance="negative"
                        className="unlink-provider-btn has-icon"
                        confirmationModalProps={{
                          title: "Disconnect Account?",
                          confirmButtonLabel: "Disconnect",
                          onConfirm: () => void handleSubmit("unlink", id),
                          children: (
                            <>
                              <p className="u-no-margin--bottom">
                                You&apos;re about to disconnect your {label}{" "}
                                account from this profile.
                              </p>
                              <p className="u-no-margin--bottom">
                                This will remove {label} as a login method.
                                Ensure you have an alternative method to access
                                your account.
                              </p>
                              <p>
                                You may reconnect it at any time in your
                                settings.
                              </p>
                            </>
                          ),
                        }}
                      >
                        <Icon name="delete" />
                        <span>Disconnect</span>
                      </ConfirmationButton>
                    )}
                  </div>
                ))}
              />
              {companySignIns.length > 0 && (
                <>
                  <p className="p-heading--5">Company sign-ins</p>
                  <p>
                    A company sign-in is attached when you sign in with your
                    company&apos;s identity provider.
                  </p>
                  <List
                    items={companySignIns.map(
                      ({ id, label, description, disabled }) => (
                        <div key={id} className="provider">
                          <img
                            src={getProviderImage(label)}
                            alt={`${label} logo`}
                            className="provider-logo"
                          />
                          <span>
                            {label}
                            {description && (
                              <>
                                <br />
                                <small className="u-text--muted">
                                  {description}
                                </small>
                              </>
                            )}
                          </span>
                          <ConfirmationButton
                            disabled={disabled}
                            appearance="negative"
                            className="unlink-provider-btn has-icon"
                            confirmationModalProps={{
                              title: "Disconnect Account?",
                              confirmButtonLabel: "Disconnect",
                              onConfirm: () =>
                                void handleCompanySignInUnlink(id),
                              children: (
                                <>
                                  <p className="u-no-margin--bottom">
                                    You&apos;re about to disconnect your {label}{" "}
                                    sign-in from this profile.
                                  </p>
                                  <p>
                                    It is attached again the next time you sign
                                    in with {label}.
                                  </p>
                                </>
                              ),
                            }}
                          >
                            <Icon name="delete" />
                            <span>Disconnect</span>
                          </ConfirmationButton>
                        </div>
                      ),
                    )}
                  />
                </>
              )}
            </>
          )}
        </div>
      )}
    </PageLayout>
  );
};

export default ManageConnectedAccounts;
