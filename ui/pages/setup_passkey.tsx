import {
  SettingsFlow,
  UpdateSettingsFlowBody,
  UpdateSettingsFlowWithWebAuthnMethod,
} from "@ory/client";
import type { NextPage } from "next";
import { useRouter } from "next/router";
import { useEffect, useState, useCallback } from "react";
import React from "react";
import { handleFlowError } from "../util/handleFlowError";
import { Flow } from "../components/Flow";
import { kratos } from "../api/kratos";
import PageLayout from "../components/PageLayout";
import { AxiosError } from "axios";
import { Spinner } from "@canonical/react-components";
import { UiNodeInputAttributes } from "@ory/client/api";
import Head from "next/head";

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
        if (data.request_url !== undefined) {
          window.location.href = `./setup_passkey?flow=${data.id}`;
          return;
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

  const handleSubmit = useCallback(
    (values: UpdateSettingsFlowBody) => {
      const methodValues = values as UpdateSettingsFlowWithWebAuthnMethod;
      console.log(values);
      return kratos
        .updateSettingsFlow({
          flow: String(flow?.id),
          updateSettingsFlowBody: {
            csrf_token: (flow?.ui?.nodes[0].attributes as UiNodeInputAttributes)
              .value as string,
            method: "webauthn",
            webauthn_register_displayname:
              methodValues.webauthn_register_displayname,
            webauthn_register: methodValues.webauthn_register,
          },
        })
        .then(async ({ data }) => {
          if (flow?.state === "success") {
            await router.push("./setup_complete");
          }
          if ("redirect_to" in data) {
            window.location.href = data.redirect_to as string;
            return;
          }
          if (flow?.return_to) {
            window.location.href = flow.return_to;
            return;
          }
          await router.push("./error");
        })
        .catch(handleFlowError("settings", setFlow));
    },
    [flow, router],
  );

  const webauthnFlow = {
    ...flow,
    ui: {
      ...flow?.ui,
      nodes: flow?.ui.nodes.filter(({ group }) => {
        return group === "webauthn";
      }),
    },
  } as SettingsFlow;

  // TODO: The webauthn script should be launched on submit
  return (
    <PageLayout title="Set up a passkey login method">
      <Head>
        <script
          src="http://localhost:4433/.well-known/ory/webauthn.js"
          type="script"
          async
        />
      </Head>
      {flow ? (
        <Flow onSubmit={handleSubmit} flow={webauthnFlow} />
      ) : (
        <Spinner />
      )}
    </PageLayout>
  );
};

export default SetupPasskey;
