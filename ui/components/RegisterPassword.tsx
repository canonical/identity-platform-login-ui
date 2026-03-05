import React from "react";
import { Button } from "@canonical/react-components";
import PageLayout from "./PageLayout";
import Password from "./Password";
import { useCallback, useMemo, useState } from "react";
import {
  RegistrationFlow,
  UiNode,
  UiNodeInputAttributes,
  UpdateRegistrationFlowBody,
} from "@ory/client";
import { kratos } from "../api/kratos";

interface RegisterPasswordProps {
  flow: RegistrationFlow | undefined;
}

export const RegisterPassword = ({ flow }: RegisterPasswordProps) => {
  const [password, setPassword] = useState("");
  const [isPassValid, setPassValid] = useState(false);

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
        .then(() => {
          // Handle successful submission, e.g., navigate to the next page
          window.location.href = "/ui/verification";
        })
        .catch((error) => {
          // Handle errors, e.g., display error messages
          console.error("Error submitting registration flow:", error);
        });
    },
    [password, CSRFToken, flow],
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
          disabled={!isPassValid}
          className="u-no-margin--bottom"
        >
          Next
        </Button>
      </form>
    </PageLayout>
  );
};
