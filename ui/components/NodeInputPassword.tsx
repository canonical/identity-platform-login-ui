import { getNodeLabel } from "@ory/integrations/ui";
import { Input } from "@canonical/react-components";
import React, { FC } from "react";
import { NodeInputProps } from "./helpers";

export const NodeInputPassword: FC<NodeInputProps> = ({
  node,
  setValue,
  disabled,
  dispatchSubmit,
  error,
}) => {
  return (
    <>
      <Input
        type="password"
        tabIndex={2}
        label={
          <>
            <span>{getNodeLabel(node)}</span>
            <a href="./reset_email" style={{ float: "right" }} tabIndex={3}>
              Reset password
            </a>
          </>
        }
        labelClassName="password-label"
        disabled={disabled}
        defaultValue={node.messages.map(({ text }) => text).join(" ")}
        error={error}
        onChange={(e) => void setValue(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "Enter") {
            e.preventDefault();
            e.stopPropagation();
            void dispatchSubmit(e, "password");
          }
        }}
      />
      <hr />
    </>
  );
};
