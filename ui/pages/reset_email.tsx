import {
  RecoveryFlow,
  UiNode,
  UiText,
  UpdateRecoveryFlowBody,
} from "@ory/client";
import type { NextPage } from "next";
import { useRouter } from "next/router";
import React, { useCallback, useEffect, useState } from "react";
import { handleFlowError } from "../util/handleFlowError";
import { Flow } from "../components/Flow";
import { kratos } from "../api/kratos";
import PageLayout from "../components/PageLayout";
import { Spinner } from "@canonical/react-components";
import {
  FlowBackButton,
  ResetEmailBackButton,
} from "../components/NavigationButtons";
import {
  isContinueWithPasswordReset,
  isResendVerificationCode,
} from "../util/constants";

const ResetEmail: NextPage = () => {
  const [flow, setFlow] = useState<RecoveryFlow>();

  // Get ?flow=... from the URL
  const router = useRouter();
  const { return_to: returnTo, flow: flowId } = router.query;

  useEffect(() => {
    // If the router is not ready yet, or we already have a flow, do nothing.
    if (!router.isReady || flow) {
      return;
    }

    // If ?flow=... was in the URL, we fetch it
    if (flowId) {
      kratos
        .getRecoveryFlow({ id: String(flowId) })
        .then((res) => setFlow(res.data))
        .catch(handleFlowError("recovery", setFlow));
      return;
    }

    // Otherwise we initialize it
    kratos
      .createBrowserRecoveryFlow({
        returnTo: returnTo ? String(returnTo) : undefined,
      })
      .then(({ data }) => {
        if (flowId !== data.id) {
          window.history.replaceState(
            null,
            "",
            `./reset_email?flow=${data.id}`,
          );
          router.query.flow = data.id;
        }
        setFlow(data);
      })
      .catch(handleFlowError("recovery", setFlow));
  }, [flowId, router, router.isReady, returnTo, flow]);

  const handleSubmit = useCallback(
    (values: UpdateRecoveryFlowBody) => {
      return kratos
        .updateRecoveryFlow({
          flow: String(flow?.id),
          updateRecoveryFlowBody: values,
        })
        .then(({ data }) => {
          if (values.email) {
            setFlow(undefined); // Trigger refresh of the flow
            return;
          }
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
        .catch(handleFlowError("recovery", setFlow));
    },
    [flow, router],
  );

  const wasEmailSent = flow?.state === "sent_email";

  const getTitle = () => {
    if (wasEmailSent) {
      return "Enter the code you received via email";
    }
    return "Enter an email to reset your password";
  };

  const withLabel = (
    node: UiNode,
    label: Partial<UiText> & { context?: object },
  ): UiNode => ({
    ...node,
    meta: {
      ...node.meta,
      label: {
        ...node.meta.label,
        ...label,
        context: { ...node.meta.label?.context, ...label.context },
      } as UiText,
    },
  });

  const getRenderFlow = (): RecoveryFlow | undefined => {
    if (!flow) {
      return flow;
    }

    if (!wasEmailSent) {
      return {
        ...flow,
        ui: {
          ...flow.ui,
          nodes: flow.ui.nodes.map((node) =>
            isContinueWithPasswordReset(node)
              ? withLabel(node, {
                  text: "Reset password",
                  context: { beforeComponent: <ResetEmailBackButton /> },
                })
              : node,
          ),
        },
      };
    }

    // Code screen: Back and "Resend code" as secondary actions, followed by
    // "Reset password" as the main action, consistent with the email screen.
    const resendNode = flow.ui.nodes.find(isResendVerificationCode);
    const nodes = flow.ui.nodes
      .filter((node) => node !== resendNode)
      .flatMap((node) => {
        if (!isContinueWithPasswordReset(node)) {
          return [node];
        }
        const submitNode = withLabel(node, { text: "Reset password" });
        const backButton = <FlowBackButton setFlow={setFlow} tabIndex={3} />;
        if (!resendNode) {
          return [
            withLabel(submitNode, { context: { beforeComponent: backButton } }),
          ];
        }
        return [
          withLabel(resendNode, {
            context: { appearance: "", beforeComponent: backButton },
          }),
          submitNode,
        ];
      });

    return { ...flow, ui: { ...flow.ui, nodes } };
  };

  return (
    <PageLayout title={getTitle()}>
      {flow ? (
        <Flow onSubmit={handleSubmit} flow={getRenderFlow()} />
      ) : (
        <Spinner />
      )}
    </PageLayout>
  );
};

export default ResetEmail;
