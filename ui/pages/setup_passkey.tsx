import { SettingsFlow, UpdateSettingsFlowBody } from "@ory/client";
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

  const handleSubmit = (values: UpdateSettingsFlowBody) => {
    // this is handled by the webauthn script
    console.log(values);
    return Promise.resolve();
  };

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
