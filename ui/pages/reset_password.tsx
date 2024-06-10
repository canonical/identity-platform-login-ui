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

const ResetPassword: NextPage = () => {
  const [password, setPassword] = React.useState("");
  const [isPassValid, setPassValid] = React.useState(false);
  const [flow, setFlow] = useState<SettingsFlow>();

  // Get ?flow=... from the URL
  const router = useRouter();
  const { flow: flowId } = router.query;

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

    // Otherwise we go back to the previous step of the reset flow
    window.location.href = "/ui/reset_email";
  }, [flowId, router, router.isReady, flow]);
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
          await router.push("/ui/reset_complete");
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
