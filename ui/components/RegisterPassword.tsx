import React from "react";
import { Button } from "@canonical/react-components";
import PageLayout from "./PageLayout";
import Password from "./Password";
import { useCallback, useMemo, useState } from "react";
import {
  ContinueWith,
  ContinueWithVerificationUi,
  RegistrationFlow,
  UiNode,
  UiNodeInputAttributes,
  UpdateRegistrationFlowBody,
} from "@ory/client";
import { AxiosError } from "axios";
import { kratos } from "../api/kratos";
import { redirectTo } from "../util/redirectTo";
import { useRouter } from "next/router";
import { handleFlowError } from "../util/handleFlowError";

interface RegisterPasswordProps {
  flow: RegistrationFlow | undefined;
  setFlow: React.Dispatch<React.SetStateAction<RegistrationFlow | undefined>>;
}

export const RegisterPassword = ({ flow, setFlow }: RegisterPasswordProps) => {
  const [password, setPassword] = useState("");
  const [isPassValid, setPassValid] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const router = useRouter();

  const CSRFToken = useMemo(() => {
    if (!flow) return "";
    const csrfNode = flow?.ui.nodes.find(
      (node: UiNode) =>
        (node.attributes as UiNodeInputAttributes).name === "csrf_token",
    );
    return csrfNode
      ? ((csrfNode.attributes as UiNodeInputAttributes).value as string)
      : "";
  }, [flow]);

  const onSubmit = useCallback(
    (e: React.FormEvent<HTMLFormElement>) => {
      if (!flow) return;
      e.preventDefault();
      setIsSubmitting(true);
      const emailNode: UiNode = flow?.ui.nodes.find(
        (node: UiNode) =>
          (node.attributes as UiNodeInputAttributes).name === "traits.email",
      ) as UiNode;
      const email = emailNode
        ? ((emailNode.attributes as UiNodeInputAttributes).value as string)
        : "";
      const values: UpdateRegistrationFlowBody = {
        method: "password",
        password: password,
        csrf_token: CSRFToken,
        traits: {
          email: email,
        },
      };
      kratos
        .updateRegistrationFlow({
          flow: flow.id,
          updateRegistrationFlowBody: values,
        })
        .then((response) => {
          const result = response.data;
          console.log(result);
          if (result.continue_with && Array.isArray(result.continue_with)) {
            const verificationAction: ContinueWith | undefined =
              result.continue_with.find(
                (item) => item.action === "show_verification_ui",
              );

            if (verificationAction) {
              const url =
                (verificationAction as ContinueWithVerificationUi).flow.url ??
                "";
              redirectTo(url, router);
              return;
            }

            // fallback to redirect_browser_to
            const redirectAction: ContinueWith | undefined =
              result.continue_with.find(
                (item) => item.action === "redirect_browser_to",
              );

            if (redirectAction) {
              redirectTo(redirectAction.redirect_browser_to, router);
              return;
            }
          }
        })
        .catch(handleFlowError("registration", setFlow))
        .catch((error: AxiosError<RegistrationFlow>) => {
          // Handle errors, e.g., display error messages
          console.error("Error submitting registration flow:", error);
          setIsSubmitting(false);
          if (error.response?.status === 400) {
            setFlow(error.response.data);
            return;
          }
          return Promise.reject(error);
        });
    },
    [password, CSRFToken, flow, router],
  );

  return (
    <PageLayout title="Create a password">
      <form onSubmit={onSubmit}>
        <input type="hidden" name="csrf_token" value={CSRFToken} />
        <Password
          checks={["lowercase", "uppercase", "number", "length"]}
          password={password}
          setPassword={setPassword}
          isValid={isPassValid}
          setValid={setPassValid}
        />
        <Button
          type="submit"
          appearance="positive"
          disabled={!isPassValid || isSubmitting}
          className="u-no-margin--bottom"
        >
          Next
        </Button>
      </form>
    </PageLayout>
  );
};
