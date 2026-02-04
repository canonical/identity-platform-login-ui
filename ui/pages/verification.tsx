import { NextPage } from "next";
import React, { useCallback, useEffect, useMemo, useState } from "react";
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
import { EmailVerificationPrompt } from "../components/EmailVerificationPrompt";
import { isResendVerificationCode } from "../util/constants";


const Verification: NextPage = () => {
  const UiNodePredicate = (node: UiNode) =>
    node.group === "code" &&
    node.type === "input" &&
    (node.attributes as UiNodeInputAttributes).name === "code";
  const [flow, setFlow] = useState<VerificationFlow>();
  const router = useRouter();
  const {
    return_to: returnTo,
    flow: flowId,
    code: verificationCode,
  } = router.query;
  
  const RESEND_CODE_TIMEOUT = 10000; // 10 seconds

  const [resendDisabled, setResendDisabled] = useState<boolean>(false);
  const disableButtonWithTimeout = () => {
    setResendDisabled(true);
    const timer = setTimeout(() => {
      setResendDisabled(false);
    }, RESEND_CODE_TIMEOUT);
    return () => clearTimeout(timer);
  };
  
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
            const codeUiNode = data.ui.nodes.find(UiNodePredicate);
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
            const continue_with: {
              action: string;
              redirect_browser_to: string;
            }[] = (
              data as {
                continue_with: Array<{
                  action: string;
                  redirect_browser_to: string;
                }>;
              }
            ).continue_with;
            if (continue_with[0].action === "redirect_browser_to") {
              window.location.href = continue_with[0].redirect_browser_to;
            }
            return;
          }
          if (
            data.state === "sent_email" &&
            data.ui.messages?.find((msg) => msg.type === "error") === undefined
          ) {
            // Check if email is sent and there is no error message
            // If no error message, add success message and disable resend button for 10 seconds
            const codeUiNode = data.ui.nodes.find(UiNodePredicate);
            if (codeUiNode) {
              codeUiNode.messages = [
                ...codeUiNode.messages,
                {
                  id: 11,
                  type: "success",
                  text: "Code sent. You can request a new one in 00:10s",
                },
              ];
            }
            // Disable resend button for 10 seconds
            disableButtonWithTimeout();
          } else if (data.ui.messages?.find((msg) => msg.type === "error")) {
            const codeUiNode = data.ui.nodes.find(UiNodePredicate);
            data.ui.messages?.forEach((message) => {
              if (message.type === "error") {
                codeUiNode?.messages.push({
                  id: message.id,
                  type: "error",
                  text: "Verification code incorrect. Check your email or resend the code.",
                });
              }
            });
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

  const userEmail = useMemo(() => {
    if (!flow) {
      return "";
    }
    const emailNode = flow.ui.nodes.find(
      (node) => (node.attributes as UiNodeInputAttributes).name === "email",
    );
    return emailNode ? (emailNode.attributes as UiNodeInputAttributes).value : "";
  }, [flow]);

  const lookupFlow = useMemo(() => {
    if (!flow) {
      return flow;
    }
    return {
      ...flow,
      ui: {
        ...flow.ui,
        nodes: flow.ui.nodes.map((node) => {
          if (
            node.group === "code" &&
            node.type === "input" &&
            (node.attributes as UiNodeInputAttributes).name === "code"
          ) {
            if (node.meta.label) {
              node.meta.label.context = {
                ...node.meta.label.context,
                beforeComponent: <EmailVerificationPrompt email={userEmail} />,
              };
            }
          }
          if (isResendVerificationCode(node)) {
            node.meta.label.context = {
              ...node.meta.label.context,
              appearance: "link",
            };
            (node.attributes as UiNodeInputAttributes).disabled = resendDisabled;
          }
          return node;
        }),
      },
    };
  }, [flow, resendDisabled]);

  if (!flow) {
    return null;
  }

  return (
    <PageLayout title="Check your email">
      {flow ? <Flow onSubmit={handleSubmit} flow={lookupFlow} /> : <Spinner />}
    </PageLayout>
  );
};

export default Verification;
