import {
  RegistrationFlow,
  UiNodeInputAttributes,
  UpdateRegistrationFlowBody,
} from "@ory/client";
import { Spinner } from "@canonical/react-components";
import { AxiosError } from "axios";
import type { NextPage } from "next";
import { useRouter } from "next/router";
import { useEffect, useState, useCallback } from "react";
import React from "react";
import { handleFlowError } from "../util/handleFlowError";
import { Flow } from "../components/Flow";
import { kratos } from "../api/kratos";
import PageLayout from "../components/PageLayout";
import {
  UpdateRegistrationFlowWithOidcMethod,
  UpdateRegistrationFlowWithPasswordMethod,
  UpdateRegistrationFlowWithProfileMethod,
} from "@ory/client/api";
import { isSignInEmailInput, isSignInWithPassword } from "../util/constants";

const Register: NextPage = () => {
  const [flow, setFlow] = useState<RegistrationFlow>();
  const router = useRouter();
  const { return_to: returnTo, flow: flowId } = router.query;

  const redirectToErrorPage = () => {
    const idParam = flowId ? `?id=${flowId.toString()}` : "";
    window.location.href = `./error${idParam}`;
  };

  useEffect(() => {
    if (!router.isReady || flow) {
      return;
    }

    if (flowId) {
      kratos
        .getRegistrationFlow({ id: String(flowId) })
        .then((res) => setFlow(res.data))
        .catch(handleFlowError("registration", setFlow))
        .catch(redirectToErrorPage);
      return;
    }

    kratos
      .createBrowserRegistrationFlow({
        returnTo: returnTo ? String(returnTo) : undefined,
      })
      .then(({ data }) => {
        if ("redirect_to" in data && typeof data.redirect_to === "string") {
          window.location.href = data.redirect_to;
          return;
        }
        setFlow(data);
      })
      .catch(handleFlowError("registration", setFlow))
      .catch(redirectToErrorPage);
  }, [flowId, router, router.isReady, returnTo, flow]);

  const handleSubmit = useCallback(
    (values: UpdateRegistrationFlowBody) => {
      let method: "oidc" | "password" | "profile";

      if ((values as UpdateRegistrationFlowWithOidcMethod).provider) {
        method = "oidc";
      } else if (
        (values as UpdateRegistrationFlowWithPasswordMethod).password
      ) {
        method = "password";
      } else {
        method = "profile";
      }

      let body: UpdateRegistrationFlowBody;

      if (method === "oidc") {
        body = {
          ...(values as UpdateRegistrationFlowWithOidcMethod),
          method: "oidc",
        };
      } else if (method === "password") {
        body = {
          ...(values as UpdateRegistrationFlowWithPasswordMethod),
          method: "password",
        };
      } else {
        // Only include traits if present
        const profileValues = values as UpdateRegistrationFlowWithProfileMethod;
        body = {
          ...profileValues,
          method: "profile",
          ...(profileValues.traits ? { traits: profileValues.traits } : {}),
        };
      }

      return kratos
        .updateRegistrationFlow({
          flow: String(flow?.id),
          updateRegistrationFlowBody: body,
        })
        .then(({ data }) => {
          if ("redirect_to" in data) {
            window.location.href = data.redirect_to as string;
            return;
          }
          if (flow?.return_to) {
            window.location.href = flow.return_to;
            return;
          }
        })
        .catch(handleFlowError("registration", setFlow))
        .catch((err: AxiosError<RegistrationFlow>) => {
          if (err.response?.status === 400) {
            setFlow(err.response.data);
            return;
          }
          return Promise.reject(err);
        });
    },
    [flow, router],
  );

  if (!flow) {
    return null;
  }

  // Adjust labels for registration
  flow.ui.nodes.map((node) => {
    if (isSignInWithPassword(node)) {
      node.meta.label.text = "Password";
    }
    if (isSignInEmailInput(node)) {
      node.meta.label.text = "Email";
    }
    return node;
  });

  // Automatically forward to single OIDC provider if only option
  const isSingleOidcOption =
    flow?.ui.nodes.length === 2 &&
    flow?.ui.nodes[1].group === "oidc" &&
    (flow?.ui.nodes[0].attributes as UiNodeInputAttributes).name ===
      "csrf_token";

  if (isSingleOidcOption) {
    const csrfNode = flow?.ui.nodes[0];
    const oidcNode = flow?.ui.nodes[1];
    const oidcAttributes = oidcNode.attributes as UiNodeInputAttributes;
    const csrfAttributes = csrfNode.attributes as UiNodeInputAttributes;
    void handleSubmit({
      method: "oidc",
      provider: oidcAttributes.value as string,
      csrf_token: csrfAttributes.value as string,
    });
  }

  return (
    <PageLayout title="Create an account">
      {isSingleOidcOption ? (
        <p className="u-text--muted">
          <Spinner style={{ marginRight: "0.5rem" }} />
          You will be redirected to the identity provider.
        </p>
      ) : flow ? (
        <Flow onSubmit={handleSubmit} flow={flow} />
      ) : (
        <Spinner />
      )}
    </PageLayout>
  );
};

export default Register;
