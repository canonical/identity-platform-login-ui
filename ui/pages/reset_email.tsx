import { RecoveryFlow, UpdateRecoveryFlowBody } from "@ory/client";
import type { NextPage } from "next";
import { useRouter } from "next/router";
import { useEffect, useState, useCallback } from "react";
import React from "react";
import { handleFlowError } from "../util/handleFlowError";
import { Flow } from "../components/Flow";
import { kratos } from "../api/kratos";
import PageLayout from "../components/PageLayout";
import { Spinner } from "@canonical/react-components";

const ResetEmail: NextPage = () => {
  const [flow, setFlow] = useState<RecoveryFlow>();

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
        .getRecoveryFlow({ id: String(flowId) })
        .then((res) => setFlow(res.data))
        .catch(handleFlowError("recovery", setFlow));
      return;
    }

    // Otherwise we initialize it
    kratos
      .createBrowserRecoveryFlow({
        returnTo: returnTo ? String(returnTo) : undefined,
      })
      .then(async ({ data }) => {
        if (data.request_url !== undefined) {
          await router.push(
            {
              pathname: "reset_email",
              query: { flow: data.id },
            },
            `${data.return_to}?flow=${data.id}`,
            { shallow: true },
          );
          return;
        }
        setFlow(data);
      })
      .catch(handleFlowError("recovery", setFlow));
  }, [flowId, router, router.isReady, returnTo, flow]);

  const handleSubmit = useCallback(
    (values: UpdateRecoveryFlowBody) => {
      return kratos
        .updateRecoveryFlow({
          flow: String(flow?.id),
          updateRecoveryFlowBody: values,
        })
        .then(({ data }) => {
          if (values.email) {
            setFlow(undefined); // Trigger refresh of the flow
            return;
          }
          if ("redirect_to" in data) {
            window.location.href = data.redirect_to as string;
            return;
          }
          if (flow?.return_to) {
            window.location.href = flow.return_to;
            return;
          }
          window.location.href = "./error";
        })
        .catch(handleFlowError("recovery", setFlow));
    },
    [flow, router],
  );

  const getTitle = () => {
    if (flow?.state === "sent_email") {
      return "Enter the code you received via email";
    }
    return "Enter an email to reset your password";
  };

  return (
    <PageLayout title={getTitle()}>
      {flow ? <Flow onSubmit={handleSubmit} flow={flow} /> : <Spinner />}
    </PageLayout>
  );
};

export default ResetEmail;
