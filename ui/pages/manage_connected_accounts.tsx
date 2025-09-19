import type { NextPage } from "next";
import React, { useCallback, useEffect, useMemo, useState } from "react";
import { useRouter } from "next/router";
import {
  Button,
  ConfirmationButton,
  Spinner,
} from "@canonical/react-components";
import { SettingsFlow, UiNodeInputAttributes } from "@ory/client";
import PageLayout from "../components/PageLayout";
import { kratos } from "../api/kratos";
import { handleFlowError } from "../util/handleFlowError";
import { AxiosError } from "axios";
import { getLoggedInName } from "../util/selfServeHelpers";
import { List, Icon, useToastNotification } from "@canonical/react-components";
import { getProviderImage } from "../util/logos";

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

const ManageConnectedAccounts: NextPage = () => {
  const [flow, setFlow] = useState<SettingsFlow>();

  const router = useRouter();
  const toastNotify = useToastNotification();
  const { return_to: returnTo } = router.query;

  const userName = getLoggedInName(flow);
  const providers = useMemo(() => buildOidcProviderStates(flow), [flow]);

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
      .catch(async (err: AxiosError<string>) => {
        if (err.response?.data.trim() === "Failed to create settings flow") {
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

  const connectionState = useMemo<ConnectionState>(() => {
    if (!providers.length) return "none";
    return providers.some((p) => p.name === "unlink")
      ? "someConnected"
      : "allDisconnected";
  }, [providers]);

  if (!flow) {
    return <Spinner />;
  }

  return (
    <PageLayout title="Connected accounts" isSelfServe={true} user={userName}>
      {flow && (
        <div>
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
            </>
          )}
        </div>
      )}
    </PageLayout>
  );
};

export default ManageConnectedAccounts;
