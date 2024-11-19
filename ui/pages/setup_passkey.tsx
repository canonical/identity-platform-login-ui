import {
  SettingsFlow,
  UiNode,
  UiNodeInputAttributes,
  UpdateSettingsFlowBody,
} from "@ory/client";
import type { NextPage } from "next";
import { useRouter } from "next/router";
import { useEffect, useState } from "react";
import React from "react";
import { handleFlowError } from "../util/handleFlowError";
import { Flow } from "../components/Flow";
import { kratos } from "../api/kratos";
import PageLayout from "../components/PageLayout";
import { AxiosError } from "axios";
import { Button, Icon, Spinner } from "@canonical/react-components";
import { UpdateSettingsFlowWithWebAuthnMethod } from "@ory/client/api";
import {
  ORY_LABEL_ID_NAME_OF_THE_SECURITY_KEY,
  ORY_LABEL_ID_REMOVE_SECURITY_ID,
} from "../util/constants";

const SetupPasskey: NextPage = () => {
  const [flow, setFlow] = useState<SettingsFlow>();
  const [loadingKeysFlow, setLoadingKeysFlow] = useState<SettingsFlow>();

  // Get ?flow=... from the URL
  const router = useRouter();
  const { return_to: returnTo, flow: flowId } = router.query;

  useEffect(() => {
    // If the router is not ready yet, or we already have a flow, do nothing.
    if (!router.isReady || flow) {
      return;
    }

    // If ?flow=... was in the URL, we fetch it
    if (flowId) {
      kratos
        .getSettingsFlow({ id: String(flowId) })
        .then((res) => {
          setFlow(res.data);
          setLoadingKeysFlow(undefined);
        })
        .catch(handleFlowError("settings", setFlow));
      return;
    }

    // Otherwise we initialize it
    kratos
      .createBrowserSettingsFlow({
        returnTo: returnTo ? String(returnTo) : undefined,
      })
      .then(({ data }) => {
        if (flowId !== data.id) {
          window.history.replaceState(
            null,
            "",
            `./setup_passkey?flow=${data.id}`,
          );
          router.query.flow = data.id;
        }
        setFlow(data);
      })
      .catch(handleFlowError("settings", setFlow))
      .catch(async (err: AxiosError<string>) => {
        if (err.response?.data.trim() === "Failed to create settings flow") {
          window.location.href = `./login?return_to=${window.location.pathname}`;
          return;
        }

        return Promise.reject(err);
      });
  }, [flowId, router, router.isReady, returnTo, flow]);

  const handleSubmit = (values: UpdateSettingsFlowBody) => {
    // adding a new key is handled by NodeInputButton that triggers the webauthn script
    // here we only handle removal of a key
    const authValues = values as UpdateSettingsFlowWithWebAuthnMethod;
    if (authValues.webauthn_remove) {
      return kratos
        .updateSettingsFlow({
          flow: String(flow?.id),
          updateSettingsFlowBody: {
            csrf_token: (flow?.ui?.nodes[0].attributes as UiNodeInputAttributes)
              .value as string,
            method: "webauthn",
            webauthn_remove: authValues.webauthn_remove,
          },
        })
        .then(() => {
          setFlow(undefined);
          setLoadingKeysFlow(flow);
        })
        .catch(handleFlowError("settings", setFlow));
    }
    return Promise.resolve();
  };

  const renderFlow = flow ?? loadingKeysFlow;
  const existingKeys: UiNode[] = [];
  const existingKeyNames: string[] = [];

  const webauthnFlow = {
    ...renderFlow,
    ui: {
      ...renderFlow?.ui,
      nodes: renderFlow?.ui.nodes
        .filter((node) => {
          if (node.meta.label?.id === ORY_LABEL_ID_REMOVE_SECURITY_ID) {
            existingKeys.push(node);
            if (
              node.meta.label?.context &&
              "display_name" in node.meta.label.context
            ) {
              existingKeyNames.push(
                node.meta.label.context.display_name as string,
              );
            }
            return false;
          }
          return node.group === "webauthn" || node.group === "default";
        })
        .map((node) => {
          if (node.meta.label?.id === ORY_LABEL_ID_NAME_OF_THE_SECURITY_KEY) {
            node.meta.label.text = "Security key name";
            node.meta.label.context = {
              ...node.meta.label.context,
              deduplicateValues: existingKeyNames,
            };
          }
          return node;
        }),
    },
  } as SettingsFlow;

  return (
    <PageLayout title="Add a security key">
      {webauthnFlow ? (
        <>
          <Flow onSubmit={handleSubmit} flow={webauthnFlow} />
          {existingKeys.length > 0 && (
            <>
              <h2 className="p-heading--4">Security keys</h2>

              {loadingKeysFlow && <Spinner />}
              {!loadingKeysFlow && (
                <>
                  {existingKeys.map((item, k) => {
                    let displayName = "";
                    if (
                      item.meta.label?.context &&
                      "display_name" in item.meta.label.context
                    ) {
                      displayName = item.meta.label.context
                        .display_name as string;
                    }
                    let addedAt = "";
                    if (
                      item.meta.label?.context &&
                      "added_at_unix" in item.meta.label.context
                    ) {
                      const date = new Date(
                        (item.meta.label.context.added_at_unix as number) *
                          1000,
                      );
                      addedAt = date.toLocaleDateString("en-US", {
                        year: "numeric",
                        month: "short",
                        day: "2-digit",
                      });
                    }

                    return (
                      <div className="security-key-list" key={k}>
                        <div className="name">{displayName}</div>
                        <div className="u-text--muted u-hide--small created">
                          Created {addedAt}
                        </div>
                        <div className="delete">
                          <Button
                            type="button"
                            className="u-no-margin--bottom"
                            hasIcon
                            onClick={() => {
                              void handleSubmit({
                                method: "webauthn",
                                webauthn_remove: (
                                  item.attributes as UiNodeInputAttributes
                                )?.value as string,
                              });
                            }}
                          >
                            <Icon name="delete" />
                            <span>Delete</span>
                          </Button>
                        </div>
                      </div>
                    );
                  })}
                </>
              )}
            </>
          )}
        </>
      ) : (
        <Spinner />
      )}
    </PageLayout>
  );
};

export default SetupPasskey;
