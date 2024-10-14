import { RecoveryFlow, UpdateRecoveryFlowBody } from "@ory/client";
import type { NextPage } from "next";
import { useRouter } from "next/router";
import React, { useCallback, useEffect, useState } from "react";
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
      .then(({ data }) => {
        if (flowId !== data.id) {
          window.history.replaceState(
            null,
            "",
            `./reset_email?flow=${data.id}`,
          );
          router.query.flow = data.id;
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

  const wasEmailSent = flow?.state === "sent_email";

  const getTitle = () => {
    if (wasEmailSent) {
      return "Enter the code you received via email";
    }
    return "Enter an email to reset your password";
  };

  const getRenderFlow = (): RecoveryFlow | undefined => {
    if (!flow) {
      return flow;
    }
    const isEnterEmail = flow.ui.nodes.length === 3;

    return {
      ...flow,
      ui: {
        ...flow.ui,
        nodes: flow.ui.nodes.map((node) => {
          if (isEnterEmail && node.meta.label?.text === "Submit") {
            return {
              ...node,
              meta: {
                ...node.meta,
                label: {
                  ...node.meta.label,
                  text: "Reset password",
                },
              },
            };
          } else {
            return node;
          }
        }),
      },
    } as RecoveryFlow;
  };

  return (
    <PageLayout title={getTitle()}>
      {flow ? (
        <Flow onSubmit={handleSubmit} flow={getRenderFlow()} />
      ) : (
        <Spinner />
      )}
    </PageLayout>
  );
};

export default ResetEmail;
