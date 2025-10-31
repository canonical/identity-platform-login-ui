/* eslint-disable */

import { RegistrationFlow, UpdateRegistrationFlowBody } from "@ory/client";
import { Spinner } from "@canonical/react-components";
import { AxiosError } from "axios";
import type { NextPage } from "next";
import { useRouter } from "next/router";
import React, { useCallback, useEffect, useState } from "react";
import { handleFlowError } from "../util/handleFlowError";
import { Flow } from "../components/Flow";
import { kratos } from "../api/kratos";
import PageLayout from "../components/PageLayout";
import {
  UpdateRegistrationFlowWithOidcMethod,
  UpdateRegistrationFlowWithPasswordMethod,
  UpdateRegistrationFlowWithProfileMethod,
} from "@ory/client/api";

type FlowPreparer = (values: any) => UpdateRegistrationFlowBody;
type SupportedFlowMethods = "oidc" | "password" | "profile";
type PreparerMap = {
  [method in SupportedFlowMethods]: FlowPreparer;
};

interface Traits {
  email?: string;
  full_name?: string;
}

function buildTraits(values: any): Traits {
  return {
    email: values["traits.email"]
      ? (values["traits.email"] as string)
      : undefined,
    full_name: values["traits.full_name"]
      ? (values["traits.full_name"] as string)
      : undefined,
  };
}

const flowPreparerMap: PreparerMap = {
  oidc: (values: any): UpdateRegistrationFlowBody => ({
    ...(values as UpdateRegistrationFlowWithOidcMethod),
    method: "oidc",
  }),
  password: (values: any): UpdateRegistrationFlowBody => ({
    ...(values as UpdateRegistrationFlowWithPasswordMethod),
    method: "password",
    traits: buildTraits(values),
  }),
  profile: (values: any): UpdateRegistrationFlowBody => ({
    ...(values as UpdateRegistrationFlowWithProfileMethod),
    method: "profile",
    traits: buildTraits(values),
  }),
};

function getFlowMethod(values: any): SupportedFlowMethods {
  if ((values as UpdateRegistrationFlowWithOidcMethod).provider) {
    return "oidc";
  } else if ((values as UpdateRegistrationFlowWithPasswordMethod).password) {
    return "password";
  } else {
    return "profile";
  }
}

const Registration: NextPage = () => {
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
      const url = new URL(window.location.href);
      url.searchParams.set("flow", flowId.toString());
      window.location.href = url.toString();

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
        setFlow(data);
      })
      .catch(handleFlowError("registration", setFlow))
      .catch(redirectToErrorPage);
  }, [flowId, router, router.isReady, returnTo, flow]);

  const handleSubmit = useCallback(
    (values: UpdateRegistrationFlowBody) => {
      const method = getFlowMethod(values);
      const body: UpdateRegistrationFlowBody = flowPreparerMap[method](values);

      return kratos
        .updateRegistrationFlow({
          flow: String(flow?.id),
          updateRegistrationFlowBody: body,
        })
        .then(({ data }) => {
          if ("continue_with" in data) {
            // @ts-ignore
            const continue_with = data.continue_with[0];
            // @ts-ignore
            if (continue_with.action == "redirect_browser_to") {
              // @ts-ignore
              window.location.href = continue_with.redirect_browser_to;
            }
            return;
          }
          setFlow(data as unknown as RegistrationFlow);
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

  const noShowFields = [
    "given_name",
    "family_name",
    "middle_name",
    "nickname",
    "preferred_username",
    "profile",
    "picture",
    "website",
    "gender",
    "birthdate",
    "zoneinfo",
    "locale",
    "phone_number",
    "address"
  ].map(name => `traits.${name}`);

  flow.ui.nodes = flow.ui.nodes.filter((node) => {
    if (node.group === "webauthn")
      return false

    const name = (node.attributes as any)["name"];
    if (!name) {
      return true;
    }

    return !noShowFields.includes(name)
  })

  return (
    <PageLayout title="Create an account">
      {flow ? <Flow onSubmit={handleSubmit} flow={flow} /> : <Spinner />}
    </PageLayout>
  );
};

export default Registration;
