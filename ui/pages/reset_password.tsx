import { SettingsFlow } from "@ory/client";
import { Button, Form, Notification } from "@canonical/react-components";
import type { NextPage } from "next";
import { useRouter } from "next/router";
import { useEffect, useState, useCallback, FormEvent } from "react";
import React from "react";
import { handleFlowError } from "../util/handleFlowError";
import { kratos } from "../api/kratos";
import PageLayout from "../components/PageLayout";
import Password from "../components/Password";
import { getCsrfToken } from "../util/getCsrfNode";
import { AxiosError } from "axios";
import { FlowResponse } from "./consent";
import {
  getLoggedInName,
  hasSelfServeReturn,
  formatReturnTo,
} from "../util/selfServeHelpers";

interface Props {
  forceSelfServe?: boolean;
}

const ResetPassword: NextPage<Props> = ({ forceSelfServe }: Props) => {
  const [password, setPassword] = React.useState("");
  const [isPassValid, setPassValid] = React.useState(false);
  const [flow, setFlow] = useState<SettingsFlow>();

  // Get ?flow=... from the URL
  const router = useRouter();
  const {
    return_to: returnTo,
    flow: flowId,
    pw_changed: pwChanged,
  } = router.query;

  const isSelfServe = forceSelfServe || hasSelfServeReturn(flow);
  const userName = getLoggedInName(flow);

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
        .catch(handleFlowError("settings", setFlow))
        .catch((err: AxiosError<unknown>) => {
          const result = err.response as FlowResponse | null;
          if (result?.data?.redirect_to) {
            window.location.href = result.data.redirect_to;
            return;
          }

          return Promise.reject(err);
        });
      return;
    }

    // Otherwise we initialize it
    kratos
      .createBrowserSettingsFlow({
        returnTo: formatReturnTo(returnTo, isSelfServe),
      })
      .then(({ data }) => {
        if (flowId !== data.id) {
          window.history.replaceState(
            null,
            "",
            `./reset_password?flow=${data.id}`,
          );
          router.query.flow = data.id;
        }
        setFlow(data);
      })
      .catch(handleFlowError("settings", setFlow))
      .catch((err: AxiosError<string>) => {
        if (err.response?.data.trim() === "Failed to create settings flow") {
          window.location.href = `./login?return_to=${window.location.pathname}`;
          return;
        }

        const result = err.response as unknown as FlowResponse | null;
        if (result?.data?.redirect_to) {
          window.location.href = result.data.redirect_to;
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
            csrf_token: getCsrfToken(flow?.ui?.nodes),
            method: "password",
            password: password,
          },
        })
        .then((result) => {
          const pwParam =
            result.data.state === "success" ? "pw_changed=success" : "";

          if (flow?.return_to) {
            const addendum = flow.return_to.includes("?") ? "&" : "?";
            window.location.href = `${flow.return_to}${addendum}${pwParam}`;
            return;
          }

          window.location.href = `./setup_secure?${pwParam}`;
        })
        .catch(handleFlowError("settings", setFlow));
    },
    [flow, router, password],
  );

  if (!flow) {
    return null;
  }

  return (
    <PageLayout
      title={isSelfServe ? "Change password" : "Reset password"}
      isSelfServe={isSelfServe}
      user={userName}
    >
      {pwChanged === "success" && (
        <Notification severity="positive">
          Password was changed successfully
        </Notification>
      )}
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
          {isSelfServe ? "Change password" : "Reset password"}
        </Button>
      </Form>
    </PageLayout>
  );
};

export default ResetPassword;
