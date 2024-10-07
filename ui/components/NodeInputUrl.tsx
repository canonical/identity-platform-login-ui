import { Button } from "@canonical/react-components";
import React, { FC, useCallback } from "react";
import { NodeInputProps } from "./helpers";

export const NodeInputUrl: FC<NodeInputProps> = ({ node }) => {
  const isWebauthnLogin = node.meta.label?.text === "Sign in with Security key";

  const handleWebauthnLogin = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      e.stopPropagation();

      const email =
        (document.querySelector('input[name="identifier"]') as HTMLInputElement)
          ?.value || "";

      // redirect to page with same url as current page, append webauthn and email params
      window.location.href = `${window.location.href}&webauthn=true&email=${email}`;
    },
    [node],
  );

  return (
    <Button
      onClick={isWebauthnLogin ? handleWebauthnLogin : undefined}
      className={isWebauthnLogin ? "oidc-login-button" : ""}
      tabIndex={4}
    >
      <img
        src="logos/Fallback.svg"
        alt="iconography of a key"
        style={{ marginRight: "0.5rem" }}
      />
      <span>{node.meta.label?.text}</span>
    </Button>
  );
};
