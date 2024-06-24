import { SettingsFlow } from "@ory/client";
import { Button, Form } from "@canonical/react-components";
import type { NextPage } from "next";
import { useRouter } from "next/router";
import { useEffect, useState, useCallback, FormEvent } from "react";
import React from "react";
import { handleFlowError } from "../util/handleFlowError";
import { kratos } from "../api/kratos";
import PageLayout from "../components/PageLayout";
import Password from "../components/Password";
import { UiNodeInputAttributes } from "@ory/client/api";
import { AxiosError } from "axios";

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
      .catch(handleFlowError(router, "settings", setFlow))
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
    (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault();

      kratos
        .updateSettingsFlow({
          flow: String(flow?.id),
          updateSettingsFlowBody: {
            csrf_token: (flow?.ui?.nodes[0].attributes as UiNodeInputAttributes)
              .value as string,
            method: "password",
            password: password,
          },
        })
        .then(async () => {
          await router.push("./setup_secure");
        })
        .catch(handleFlowError(router, "settings", setFlow));
    },
    [flow, router, password],
  );

  return (
    <PageLayout title="Reset password">
      <Form onSubmit={handleSubmit}>
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
        </Button>
      </Form>
    </PageLayout>
  );
};

export default ResetPassword;
