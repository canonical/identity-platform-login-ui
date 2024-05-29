import { RecoveryFlow, UpdateRecoveryFlowBody } from "@ory/client";
import { AxiosError } from "axios";
import type { NextPage } from "next";
import { useRouter } from "next/router";
import { useEffect, useState, useCallback } from "react";
import React from "react";
import { handleFlowError } from "../util/handleFlowError";
import { Flow } from "../components/Flow";
import { kratos } from "../api/kratos";
import PageLayout from "../components/PageLayout";
import { Button, Input, Spinner } from "@canonical/react-components";

const ResetComplete: NextPage = () => {
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
        .catch(handleFlowError(router, "recovery", setFlow));
      return;
    }

    // Otherwise we initialize it
    kratos
      .createBrowserRecoveryFlow({
        returnTo: returnTo ? String(returnTo) : undefined,
      })
      // .then(({ data }: FlowResponse) => {
      //   if (data.redirect_to !== undefined) {
      //     window.location.href = data.redirect_to;
      //     return;
      //   }
      //   setFlow(data);
      // })
      .catch(handleFlowError(router, "recovery", setFlow));
  }, [flowId, router, router.isReady, returnTo, flow]);
  const handleSubmit = useCallback(
    (values: UpdateRecoveryFlowBody) => {
      return kratos
        .updateRecoveryFlow({
          flow: String(flow?.id),
          updateRecoveryFlowBody: values,
        })
        .then(async ({ data }) => {
          if ("redirect_to" in data) {
            window.location.href = data.redirect_to as string;
            return;
          }
          if (flow?.return_to) {
            window.location.href = flow.return_to;
            return;
          }
          await router.push("/");
        })
        .catch(handleFlowError(router, "recovery", setFlow))
        .catch((err: AxiosError<RecoveryFlow>) => {
          if (err.response?.status === 400) {
            setFlow(err.response.data);
            return;
          }

          return Promise.reject(err);
        });
    },
    [flow, router],
  );

  return (
    <PageLayout title="Enter an email to reset your password">
      <Input
        id="email"
        name="email"
        type="text"
        label="Email"
        placeholder="Your email"
      />
      <Button type="button" className="u-no-margin--bottom">
        Back
      </Button>
      <Button
        type="submit"
        appearance="positive"
        className="u-no-margin--bottom"
      >
        Reset password
        {flow ? <Flow onSubmit={handleSubmit} flow={flow} /> : <Spinner />}
      </Button>
    </PageLayout>
  );
};

export default ResetComplete;
