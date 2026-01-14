import { NextPage } from "next";
import React, { useCallback, useEffect, useState } from "react";
import {
  UiNode,
  UiNodeInputAttributes,
  UpdateVerificationFlowBody,
  UpdateVerificationFlowWithCodeMethod,
  VerificationFlow,
} from "@ory/client";
import { useRouter } from "next/router";
import { handleFlowError } from "../util/handleFlowError";
import { kratos } from "../api/kratos";
import { Flow } from "../components/Flow";
import PageLayout from "../components/PageLayout";
import { Spinner } from "@canonical/react-components";
import { AxiosError } from "axios";
import { setFlowIDQueryParam } from "../util/flowHelper";

const Verification: NextPage = () => {
  const [flow, setFlow] = useState<VerificationFlow>();
  const router = useRouter();
  const {
    return_to: returnTo,
    flow: flowId,
    code: verificationCode,
  } = router.query;

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
        .getVerificationFlow({ id: String(flowId) })
        .then(({ data }) => {
          if (verificationCode) {
            const predicate = (node: UiNode) =>
              node.group === "code" &&
              node.type === "input" &&
              (node.attributes as UiNodeInputAttributes).name === "code";
            const codeUiNode = data.ui.nodes.find(predicate);
            if (codeUiNode) {
              (codeUiNode.attributes as UiNodeInputAttributes).value =
                String(verificationCode);
            }
          }
          setFlowIDQueryParam(router, data.id);
          setFlow(data);
        })
        .catch(handleFlowError("verification", setFlow))
        .catch(redirectToErrorPage);

      return;
    }

    kratos
      .createBrowserVerificationFlow({
        returnTo: returnTo ? String(returnTo) : undefined,
      })
      .then(({ data }) => {
        setFlow(data);
        setFlowIDQueryParam(router, String(data.id));
      })
      .catch(handleFlowError("verification", setFlow))
      .catch(redirectToErrorPage);
  }, [flowId, router, router.isReady, returnTo]);

  const handleSubmit = useCallback(
    (values: UpdateVerificationFlowBody) => {
      return kratos
        .updateVerificationFlow({
          flow: String(flow?.id),
          updateVerificationFlowBody: {
            ...(values as UpdateVerificationFlowWithCodeMethod),
            method: "code",
          },
        })
        .then(({ data }) => {
          if ("continue_with" in data) {
            const continue_with: any = (
              data as { continue_with: Array<{ action: string } & any> }
            ).continue_with[0];
            if (continue_with.action === "redirect_browser_to") {
              window.location.href = continue_with.redirect_browser_to;
            }
            return;
          }
          setFlow(data);
        })
        .catch(handleFlowError("verification", setFlow))
        .catch((err: AxiosError<VerificationFlow>) => {
          if (err.response?.status === 400) {
            setFlow(err.response.data);
            return;
          }
          return Promise.reject(err);
        });
    },
    [flow],
  );

  if (!flow) {
    return null;
  }

  return (
    <PageLayout title="Verify your email">
      {flow ? <Flow onSubmit={handleSubmit} flow={flow} /> : <Spinner />}
    </PageLayout>
  );
};

export default Verification;
