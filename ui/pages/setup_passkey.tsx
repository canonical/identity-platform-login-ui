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
import { Accordion, Button, Icon, Spinner } from "@canonical/react-components";
import { UpdateSettingsFlowWithWebAuthnMethod } from "@ory/client/api";
import {
  ORY_LABEL_ID_ADD_SECURITY_KEY,
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

  const isSequencedFromSignInFlow =
    flow?.return_to?.includes("/ui/login?") ?? false;

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
          if (node.meta.label?.id === ORY_LABEL_ID_ADD_SECURITY_KEY) {
            node.meta.label.context = {
              ...node.meta.label.context,
              appearance: isSequencedFromSignInFlow ? "" : "positive",
              additional: (
                <a
                  className="p-button--link"
                  href="mailto:support@canonical.com"
                >
                  Contact support
                </a>
              ),
            };
          }
          return node;
        }),
    },
  } as SettingsFlow;

  const highlightPasskeyRequirement = () => {
    const nameSelector = '*[name="webauthn_register_displayname"]';
    const name = document.querySelector(nameSelector) as HTMLInputElement;

    if (!name?.value) {
      name.classList.add("is-error");
      name.setAttribute("aria-invalid", "true");
      name.setAttribute(
        "aria-errormessage",
        "You must set up a security key to sign in",
      );
    }

    if (!name.nextElementSibling) {
      const warning = document.createElement("p");
      warning.className = "p-form-validation__message";
      warning.textContent = "You must set up a security key to sign in";
      name.after(warning);
    }

    const groupSelector = ".p-form-validation";
    const group = document.querySelectorAll(groupSelector);
    group.forEach((e) => e.classList.add("is-error"));
  };

  return (
    <PageLayout
      title={isSequencedFromSignInFlow ? "Sign in" : "Add a security key"}
    >
      {isSequencedFromSignInFlow && (
        <>
          <p className="u-text--muted u-sv-3">
            Authentication setup needed to continue.
          </p>
          <h2 className="p-heading--4">Add a security key</h2>
          <Accordion
            sections={[
              {
                title: "How to add a Security key",
                content: (
                  <>
                    <ol className="p-list--nested-counter">
                      <li>
                        Enter a name for your security key (like {'"'}iPhone
                        {'"'} or
                        {'"'} Work Laptop{'"'})
                      </li>
                      <li>
                        Click {'"'}Add security key{'"'}
                      </li>

                      <li>
                        When your browser prompts you, use your device{"'"}s
                        fingerprint, face recognition, or PIN to verify
                      </li>
                      <li>Wait for confirmation</li>
                    </ol>
                    <p>
                      That{"'"}s it! Your passkey is now set up for future
                      sign-ins.
                    </p>
                    <p>
                      (Note: Your device needs biometrics or a PIN already
                      configured)
                    </p>
                  </>
                ),
              },
            ]}
          />
        </>
      )}
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
          {isSequencedFromSignInFlow && (
            <>
              <hr />
              <div className="u-align--right">
                <Button
                  appearance="positive"
                  type="button"
                  className="u-no-margin--bottom u-align--right"
                  onClick={highlightPasskeyRequirement}
                >
                  Sign in
                </Button>
              </div>
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
