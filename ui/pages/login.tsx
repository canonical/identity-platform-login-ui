import { LoginFlow, UpdateLoginFlowBody } from "@ory/client";
import { Card, Row, Spinner } from "@canonical/react-components";
import { AxiosError } from "axios";
import type { NextPage } from "next";
import Head from "next/head";
import { useRouter } from "next/router";
import { useEffect, useState, useCallback } from "react";
import React from "react";
import { handleFlowError } from "../util/handleFlowError";
import { Flow } from "../components/Flow";
import { kratos } from "../api/kratos";
import { FlowResponse } from "./consent";

const Login: NextPage = () => {
  const [flow, setFlow] = useState<LoginFlow>();

  // Get ?flow=... from the URL
  const router = useRouter();
  const {
    return_to: returnTo,
    flow: flowId,
    // Refresh means we want to refresh the session. This is needed, for example, when we want to update the password
    // of a user.
    refresh,
    // AAL = Authorization Assurance Level. This implies that we want to upgrade the AAL, meaning that we want
    // to perform two-factor authentication/verification.
    aal,
    login_challenge,
  } = router.query;

  useEffect(() => {
    // If the router is not ready yet, or we already have a flow, do nothing.
    if (!router.isReady || flow) {
      return;
    }

    // If ?flow=.. was in the URL, we fetch it
    if (flowId) {
      kratos
        .getLoginFlow({ id: String(flowId) })
        .then(void setFlow)
        .catch(handleFlowError(router, "login", setFlow));
      return;
    }

    // Otherwise we initialize it
    kratos
      .createBrowserLoginFlow({
        refresh: Boolean(refresh),
        aal: aal ? String(aal) : undefined,
        returnTo: returnTo ? String(returnTo) : undefined,
        loginChallenge: login_challenge ? String(login_challenge) : undefined,
      })
      .then(({ data }: FlowResponse) => {
        if (data.redirect_to !== undefined) {
          window.location.href = data.redirect_to;
          return;
        }
        setFlow(data);
      })
      .catch(handleFlowError(router, "login", setFlow));
  }, [
    flowId,
    router,
    router.isReady,
    aal,
    refresh,
    returnTo,
    flow,
    login_challenge,
  ]);
  const handleSubmit = useCallback(
    (values: UpdateLoginFlowBody) =>
      kratos
        .updateLoginFlow({
          flow: String(flow?.id),
          updateLoginFlowBody: values,
        })
        // We logged in successfully! Let's bring the user home.
        .then(async () => {
          if (flow?.return_to) {
            window.location.href = flow.return_to;
            return;
          }
          await router.push("/");
        })
        .catch(handleFlowError(router, "login", setFlow))
        .catch((err: AxiosError<LoginFlow>) => {
          // If the previous handler did not catch the error it's most likely a form validation error
          if (err.response?.status === 400) {
            // Yup, it is!
            setFlow(err.response.data);
            return;
          }

          return Promise.reject(err);
        }),
    [flow, router]
  );
  return (
    <>
      <Head>
        <title>Sign in</title>
      </Head>
      <Row className="p-strip">
        <div className="login-card">
          <div>
            {flow ? (
              <Card title="Choose Provider">
                <Flow onSubmit={handleSubmit} flow={flow} />
              </Card>
            ) : (
              <Spinner />
            )}
          </div>
        </div>
      </Row>
    </>
  );
};

export default Login;
