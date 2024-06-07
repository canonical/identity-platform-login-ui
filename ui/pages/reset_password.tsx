import { SettingsFlow, UpdateSettingsFlowBody } from "@ory/client";
import { AxiosError } from "axios";
import { Button, Spinner } from "@canonical/react-components";
import type { NextPage } from "next";
import { useRouter } from "next/router";
import { useEffect, useState, useCallback } from "react";
import React from "react";
import { handleFlowError } from "../util/handleFlowError";
import { Flow } from "../components/Flow";
import { kratos } from "../api/kratos";
import PageLayout from "../components/PageLayout";
import Password from "../components/Password";

const ResetPassword: NextPage = () => {
  const [password, setPassword] = React.useState("");
  const [isPassValid, setPassValid] = React.useState(false);
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
        .catch(handleFlowError(router, "settings", setFlow));
      return;
    }

    // Otherwise we initialize it
    kratos
      .createBrowserSettingsFlow({
        returnTo: returnTo ? String(returnTo) : undefined,
      })
      .then(({ data }) => {
        if (data.request_url !== undefined) {
          window.location.href = data.request_url;
          return;
        }
        setFlow(data);
      })
      .catch(handleFlowError(router, "settings", setFlow));
  }, [flowId, router, router.isReady, returnTo, flow]);
  const handleSubmit = useCallback(
    (values: UpdateSettingsFlowBody) => {
      return kratos
        .updateSettingsFlow({
          flow: String(flow?.id),
          updateSettingsFlowBody: values,
        })
        .then(async ({ data }) => {
          console.log(data); // TODO: remove
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
        .catch(handleFlowError(router, "settings", setFlow))
        .catch((err: AxiosError<SettingsFlow>) => {
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
    <PageLayout title="Reset password">
      <Password
        checks={["lowercase", "uppercase", "number", "length"]}
        password={password}
        setPassword={setPassword}
        isValid={isPassValid}
        setValid={setPassValid}
        label="New password"
      />
      <Button
        type="submit"
        appearance="positive"
        disabled={!isPassValid}
        className="u-no-margin--bottom"
      >
        Reset password
        {flow ? <Flow onSubmit={handleSubmit} flow={flow} /> : <Spinner />}
      </Button>
    </PageLayout>
  );
};

export default ResetPassword;
