import { Button } from "@canonical/react-components";
import PageLayout from "./PageLayout";
import Password from "./Password";
import { useCallback, useContext, useMemo, useState } from "react";
import { UiNode, UiNodeInputAttributes, UpdateRegistrationFlowBody } from "@ory/client";
import { redirect } from "next/navigation";
import { kratos } from "../api/kratos";

interface RegisterPasswordProps {
  flow: any;
}

export const RegisterPassword = ({
  flow
}: RegisterPasswordProps) => {
  const [password, setPassword] = useState("");
  const [isPassValid, setPassValid] = useState(false);

  const CSRFToken = useMemo(() => {
    if (!flow) return "";
    const csrfNode  = flow?.ui.nodes.find(
      (node: UiNode) => (node.attributes as UiNodeInputAttributes).name === "csrf_token",
    );
    return csrfNode ? ((csrfNode.attributes as UiNodeInputAttributes).value as string) : "";
  }, [flow]);

  const onSubmit = useCallback((e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const values: UpdateRegistrationFlowBody = {
      method: "password",
      password: password,
      csrf_token: CSRFToken,
      traits: {
        email: flow?.ui.nodes.find(
          (node: UiNode) => (node.attributes as UiNodeInputAttributes).name === "traits.email",
        )?.attributes.value as string,
      },
    }
    kratos.updateRegistrationFlow({
      flow: flow.id,
      updateRegistrationFlowBody: values,
    })
    .then(() => {
      // Handle successful submission, e.g., navigate to the next page
      window.location.href = "/ui/verification";
    }).catch((error) => {
      // Handle errors, e.g., display error messages
      console.error("Error submitting registration flow:", error);
    });
  }, [password, CSRFToken, flow]);

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