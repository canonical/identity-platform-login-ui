import {RegistrationFlow, UpdateRegistrationFlowBody,} from "@ory/client";
import {Spinner} from "@canonical/react-components";
import {AxiosError} from "axios";
import type {NextPage} from "next";
import {useRouter} from "next/router";
import React, {useCallback, useEffect, useState} from "react";
import {handleFlowError} from "../util/handleFlowError";
import {Flow} from "../components/Flow";
import {kratos} from "../api/kratos";
import PageLayout from "../components/PageLayout";
import {
  UpdateRegistrationFlowWithOidcMethod,
  UpdateRegistrationFlowWithPasswordMethod,
  UpdateRegistrationFlowWithProfileMethod,
} from "@ory/client/api";

type FlowPreparer = (values: any) => UpdateRegistrationFlowBody;
type SupportedFlowMethods = "oidc" | "password" | "profile";
type PreparerMap = {
  [method in SupportedFlowMethods]: FlowPreparer
};

interface Traits {
  email?: string;
  full_name?: string;
}

function buildTraits(values: any): Traits {
  return {
    email: values["traits.email"] ? values["traits.email"] as string : undefined,
    full_name: values["traits.full_name"] ? values["traits.full_name"] as string : undefined,
  }
}

const flowPreparerMap: PreparerMap = {
  "oidc": (values: any): UpdateRegistrationFlowBody => ({
    ...(values as UpdateRegistrationFlowWithOidcMethod),
    method: "oidc"
  }),
  "password": (values: any): UpdateRegistrationFlowBody => ({
    ...(values as UpdateRegistrationFlowWithPasswordMethod),
    method: "password",
    traits: buildTraits(values)
  }),
  "profile": (values: any): UpdateRegistrationFlowBody => (
    {
      ...values as UpdateRegistrationFlowWithProfileMethod,
      method: "profile",
      traits: buildTraits(values)
    }
  )
}

function getFlowMethod(values: any): SupportedFlowMethods {
  if ((values as UpdateRegistrationFlowWithOidcMethod).provider) {
    return "oidc";
  } else if (
    (values as UpdateRegistrationFlowWithPasswordMethod).password
  ) {
    return "password";
  } else {
    return "profile";
  }
}

const Registration: NextPage = () => {
  const [flow, setFlow] = useState<RegistrationFlow>();
  const router = useRouter();
  const {return_to: returnTo, flow: flowId} = router.query;

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
        .getRegistrationFlow({id: String(flowId)})
        .then((res) => setFlow(res.data))
        .catch(handleFlowError("registration", setFlow))
        .catch(redirectToErrorPage);

      return;
    }

    kratos
      .createBrowserRegistrationFlow({
        returnTo: returnTo ? String(returnTo) : undefined,
      })
      .then(({data}) => {
        if ("redirect_to" in data) {
          window.location.href = data.redirect_to as string;
          return;
        }

        setFlow(data);
      })
      .catch(handleFlowError("registration", setFlow))
      .catch(redirectToErrorPage);
  }, [flowId, router, router.isReady, returnTo, flow]);

  const handleSubmit = useCallback(
    (values: UpdateRegistrationFlowBody) => {
      let method = getFlowMethod(values);
      let body: UpdateRegistrationFlowBody = flowPreparerMap[method](values)

      return kratos
        .updateRegistrationFlow({
          flow: String(flow?.id),
          updateRegistrationFlowBody: body
        })
        .then(({data}) => {
          if ("redirect_to" in data) {
            window.location.href = data.redirect_to as string;
            return;
          }

          if (flow?.return_to) {
            window.location.href = flow.return_to;
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
    [flow, router]
  );

  if (!flow) {
    return null;
  }

  return (
    <PageLayout title="Create an account">
      {flow ? <Flow onSubmit={handleSubmit} flow={flow}/> : <Spinner/>}
    </PageLayout>
  );
};

export default Registration;
