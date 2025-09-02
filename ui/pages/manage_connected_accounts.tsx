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
import type {
  UpdateSettingsFlowBody,
  UpdateSettingsFlowWithOidcMethod,
} from "@ory/client";

type ProviderConnectionAction = "link" | "unlink";

type SettingsFlowWithRedirect = SettingsFlow & {
  redirect_to?: string;
};

interface ProviderState {
  id: string;
  label: string;
  name: ProviderConnectionAction;
  disabled: boolean;
}

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
  const { return_to: returnTo, flow: flowId } = router.query;

  const userName = getLoggedInName(flow);
  const providers = useMemo(() => buildOidcProviderStates(flow), [flow]);

  useEffect(() => {
    if (!router.isReady || flow) {
      return;
    }

    if (flowId) {
      kratos
        .getSettingsFlow({ id: String(flowId) })
        .then((res) => setFlow(res.data))
        .catch(handleFlowError("settings", setFlow));
      return;
    }

    kratos
      .createBrowserSettingsFlow({
        returnTo: returnTo ? String(returnTo) : undefined,
      })
      .then(({ data }) => {
        if (flowId !== data.id) {
          window.history.replaceState(
            null,
            "",
            `./manage_connected_accounts?flow=${data.id}`,
          );
          router.query.flow = data.id;
        }
        setFlow(data);
      })
      .catch(async (err: AxiosError<string>) => {
        if (err.response?.data.trim() === "Failed to create settings flow") {
          window.location.href = `./login?return_to=${window.location.pathname}`;
          return;
        }

        return Promise.reject(err);
      });
  }, [flowId, router, router.isReady, returnTo, flow]);

  const handleSubmit = useCallback(
    (values: UpdateSettingsFlowBody) => {
      const methodValues = values as UpdateSettingsFlowWithOidcMethod;
      return kratos
        .updateSettingsFlow({
          flow: String(flow?.id),
          updateSettingsFlowBody: {
            method: "oidc",
            ...(methodValues.link ? { link: methodValues.link } : {}),
            ...(methodValues.unlink ? { unlink: methodValues.unlink } : {}),
          },
        })
        .then((result) => {
          const data = result.data as SettingsFlowWithRedirect;
          if (data.redirect_to) {
            window.location.href = data.redirect_to;
            toastNotify.success(
              `Your ${methodValues.link} account is now connected and can be used to sign in.`,
              undefined,
              "Account connected successfully",
            );
            return;
          }
          setFlow(data);
          toastNotify.success(
            `Your ${methodValues.unlink} account has been disconnected.`,
            undefined,
            "Account disconnected successfully",
          );
        })
        .catch(handleFlowError("settings", setFlow));
    },
    [flow, router],
  );

  const connectionState = useMemo<
    "none" | "allDisconnected" | "someConnected"
  >(() => {
    if (!providers.length) return "none";
    return providers.some((p) => p.name === "unlink")
      ? "someConnected"
      : "allDisconnected";
  }, [providers]);

  const connectionText: Record<typeof connectionState, string> = {
    none: "No accounts to connect right now. Services will appear here once they’ve been made available to connect.",
    allDisconnected:
      "You haven’t connected any accounts yet. Connect an account from a service as another way to sign in quickly and securely.",
    someConnected:
      "You’ve connected the accounts below. Connect an account from a service as another way to sign in quickly and securely.",
  };

  if (!flow) {
    return <Spinner />;
  }

  return (
    <PageLayout title="Connected accounts" isSelfServe={true} user={userName}>
      {flow && (
        <div>
          <p>{connectionText[connectionState]}</p>
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
                        onClick={() =>
                          handleSubmit({
                            method: "oidc",
                            link: id,
                          })
                        }
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
                          onConfirm: () =>
                            void handleSubmit({
                              method: "oidc",
                              unlink: id,
                            }),
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
