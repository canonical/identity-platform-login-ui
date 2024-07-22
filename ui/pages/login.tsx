import {
  LoginFlow,
  UiNode,
  UiNodeInputAttributes,
  UpdateLoginFlowBody,
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
import { FlowResponse } from "./consent";
import PageLayout from "../components/PageLayout";
import { replaceAuthLabel } from "../util/replaceAuthLabel";
import { UpdateLoginFlowWithOidcMethod } from "@ory/client/api";

const Login: NextPage = () => {
  const [flow, setFlow] = useState<LoginFlow>();
  const isAuthCode = flow?.ui.nodes.find((node) => node.group === "totp");

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
        .then((res) => setFlow(res.data))
        .catch(handleFlowError("login", setFlow));
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
      .catch(handleFlowError("login", setFlow));
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
    (values: UpdateLoginFlowBody) => {
      const getMethod = () => {
        if ((values as UpdateLoginFlowWithOidcMethod).provider) {
          return "oidc";
        }
        if (values.method === "webauthn") {
          return "webauthn";
        }
        if (isAuthCode) {
          return "totp";
        }
        return "password";
      };

      return kratos
        .updateLoginFlow({
          flow: String(flow?.id),
          updateLoginFlowBody: {
            ...values,
            method: getMethod(),
          } as UpdateLoginFlowBody,
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
          window.location.href = "./error";
        })
        .catch(handleFlowError("login", setFlow))
        .catch((err: AxiosError<LoginFlow>) => {
          if (err.response?.status === 400) {
            setFlow(err.response.data);
            return;
          }

          return Promise.reject(err);
        });
    },
    [flow, router],
  );
  const reqName = flow?.oauth2_login_request?.client?.client_name;
  const reqDomain = flow?.oauth2_login_request?.client?.client_uri
    ? new URL(flow.oauth2_login_request.client.client_uri).hostname
    : "";
  const getTitleSuffix = () => {
    if (reqName && reqDomain) {
      return ` to ${reqName} on ${reqDomain}`;
    }
    if (reqName) {
      return ` to ${reqName}`;
    }
    if (reqDomain) {
      return ` to ${reqDomain}`;
    }
    return "";
  };
  const title = isAuthCode
    ? "Verify your identity"
    : `Sign in${getTitleSuffix()}`;
  const renderFlow = isAuthCode ? replaceAuthLabel(flow) : flow;

  let isWebauthn = false;
  if (renderFlow?.ui) {
    const urlParams = new URLSearchParams(window.location.search);
    isWebauthn =
      urlParams.get("webauthn") === "true" ||
      renderFlow.ui.nodes.filter(
        (node) => node.group !== "webauthn" && node.group !== "default",
      ).length === 0;

    renderFlow.ui.nodes = renderFlow?.ui.nodes.filter((node) => {
      // show webauthn elements in dedicated step after it is selected
      if (isWebauthn) {
        return node.group === "webauthn" || node.group === "default";
      }
      // hide webauthn everywhere else
      return node.group !== "webauthn";
    });

    // add security key option that looks like an oidc input
    if (!isWebauthn && !isAuthCode) {
      renderFlow.ui.nodes.push({
        attributes: {
          type: "url",
          node_type: "input",
          name: "",
          disabled: false,
        },
        group: "webauthn",
        type: "input",
        messages: [],
        meta: {
          label: {
            id: 1,
            text: "Sign in with Security key",
            type: "info",
          },
        },
      });
    }

    // ensure oidc options are presented after username/password inputs
    renderFlow.ui.nodes.sort((a, b) => {
      const toValue = (node: UiNode) => (node.group === "oidc" ? 1 : -1);
      return toValue(a) - toValue(b);
    });

    // autosubmit webauthn in case email is provided
    const email = urlParams.get("email");
    if (isWebauthn && email) {
      const csrfNode = renderFlow?.ui.nodes.find(
        (node) =>
          node.group === "default" &&
          node.attributes.node_type === "input" &&
          node.attributes.name === "csrf_token",
      )?.attributes as UiNodeInputAttributes;

      void handleSubmit({
        method: "webauthn",
        identifier: email,
        csrf_token: (csrfNode.value as string) ?? "",
      }).catch(() => {
        if (flow?.return_to) {
          window.location.href = flow.return_to;
        }
      });

      return null;
    }
  }

  if (!flow) {
    return;
  }

  return (
    <PageLayout title={title}>
      {flow ? <Flow onSubmit={handleSubmit} flow={renderFlow} /> : <Spinner />}
      {isWebauthn && <a href={flow?.return_to}>I want to use another method</a>}
    </PageLayout>
  );
};

export default Login;
