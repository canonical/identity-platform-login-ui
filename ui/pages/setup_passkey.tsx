import {
  SettingsFlow,
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
import { Spinner } from "@canonical/react-components";
import { UpdateSettingsFlowWithWebAuthnMethod } from "@ory/client/api";

const SetupPasskey: NextPage = () => {
  const [flow, setFlow] = useState<SettingsFlow>();

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
        .then((res) => setFlow(res.data))
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
        }
        setFlow(data);
      })
      .catch(handleFlowError("settings", setFlow))
      .catch(async (err: AxiosError<string>) => {
        if (err.response?.data.trim() === "Failed to create settings flow") {
          setFlow(undefined);
          await router.push("./login");
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
          window.location.href = "./setup_passkey";
        })
        .catch(handleFlowError("settings", setFlow));
    }
    return Promise.resolve();
  };

  const webauthnFlow = {
    ...flow,
    ui: {
      ...flow?.ui,
      nodes: flow?.ui.nodes.filter(({ group }) => {
        return group === "webauthn" || group === "default";
      }),
    },
  } as SettingsFlow;

  return (
    <PageLayout title="Set up a passkey login method">
      {flow ? (
        <Flow onSubmit={handleSubmit} flow={webauthnFlow} />
      ) : (
        <Spinner />
      )}
    </PageLayout>
  );
};

export default SetupPasskey;
