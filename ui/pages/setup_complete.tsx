import type { NextPage } from "next";
import React, { useEffect, useState } from "react";
import PageLayout from "../components/PageLayout";
import { useRouter } from "next/router";
import { Spinner } from "@canonical/react-components";
import { kratos } from "../api/kratos";
import { handleFlowError } from "../util/handleFlowError";
import { SettingsFlow } from "@ory/client";

const SetupComplete: NextPage = () => {
  const [flow, setFlow] = useState<SettingsFlow>();

  const router = useRouter();
  const { flow: flowId } = router.query;

  useEffect(() => {
    // If the router is not ready yet, or we have no flowId, do nothing.
    if (!router.isReady || !flowId) {
      return;
    }

    // If the flow has a return_to, we redirect to it after 3 seconds
    if (flow?.return_to) {
      setTimeout(() => {
        window.location.href = flow.return_to as string;
      }, 3000);
    }

    if (flow) {
      return;
    }

    // If ?flow=... was in the URL, we fetch it
    kratos
      .getSettingsFlow({ id: String(flowId) })
      .then((res) => setFlow(res.data))
      .catch(handleFlowError("settings", setFlow));
  }, [flowId, router, flow]);

  return (
    <PageLayout title="Account setup complete">
      {flow?.return_to ? (
        <>
          <p className="u-text--muted">
            <Spinner style={{ marginRight: "0.5rem" }} />
            You will be redirected back to the application.
          </p>
          <p className="u-text--muted">
            <a href={flow?.return_to}>Continue immediately</a>
          </p>
        </>
      ) : (
        <p className="u-text--muted">You can close this tab.</p>
      )}
    </PageLayout>
  );
};

export default SetupComplete;
