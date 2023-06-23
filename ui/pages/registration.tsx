import {
  LoginFlow,
  RegistrationFlow,
  UpdateRegistrationFlowBody,
} from "@ory/client";
import { Card, Row } from "@canonical/react-components";
import { AxiosError } from "axios";
import type { NextPage } from "next";
import Head from "next/head";
import { useRouter } from "next/router";
import { useEffect, useState, useCallback } from "react";
// Import render helpers
import { Flow } from "../components/Flow";
import Link from "next/link";
import { handleFlowError } from "../util/handleFlowError";
import { kratos } from "../api/kratos";
import React from "react";

// Renders the registration page
const Registration: NextPage = () => {
  const router = useRouter();

  // The "flow" represents a registration process and contains
  // information about the form we need to render (e.g. username + password)
  const [flow, setFlow] = useState<RegistrationFlow>();

  // Get ?flow=... from the URL
  const { flow: flowId, return_to: returnTo } = router.query;

  // In this effect we either initiate a new registration flow, or we fetch an existing registration flow.
  useEffect(() => {
    // If the router is not ready yet, or we already have a flow, do nothing.
    if (!router.isReady || flow) {
      return;
    }

    // If ?flow=.. was in the URL, we fetch it
    if (flowId) {
      kratos
        .getRegistrationFlow({ id: String(flowId) })
        .then(({ data }) => {
          // We received the flow - let's use its data and render the form!
          setFlow(data);
        })
        .catch(handleFlowError(router, "registration", setFlow));
      return;
    }

    // Otherwise we initialize it
    kratos
      .createBrowserRegistrationFlow({
        returnTo: returnTo ? String(returnTo) : undefined,
      })
      .then(({ data }) => {
        setFlow(data);
      })
      .catch(handleFlowError(router, "registration", setFlow));

    return;
  }, [flowId, router, router.isReady, returnTo, flow]);

  const handleSubmit = useCallback(
    (values: UpdateRegistrationFlowBody) =>
      router
        // On submission, add the flow ID to the URL but do not navigate. This prevents the user loosing
        // his data when she/he reloads the page.
        .push(`/registration?flow=${flow?.id ?? ""}`, undefined, {
          shallow: true,
        })
        .then(() =>
          kratos
            .updateRegistrationFlow({
              flow: String(flow?.id),
              updateRegistrationFlowBody: values,
            })
            .then(async () => {
              await router.push(flow?.return_to ?? "/");
            })
            .catch(handleFlowError(router, "registration", setFlow))
            .catch((err: AxiosError<LoginFlow>) => {
              // If the previous handler did not catch the error it's most likely a form validation error
              if (err.response?.status === 400) {
                // Yup, it is!
                setFlow(err.response.data);
                return;
              }

              return Promise.reject(err);
            })
        ),
    [flow, router]
  );

  return (
    <>
      <Head>
        <title>Create account</title>
      </Head>
      <Row className="p-strip">
        <Card title="Create account">
          <Flow onSubmit={handleSubmit} flow={flow} />
        </Card>
        <Card>
          <Link href="/login">Sign in</Link>
        </Card>
      </Row>
    </>
  );
};

export default Registration;
