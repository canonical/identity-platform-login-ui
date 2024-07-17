import { getNodeLabel } from "@ory/integrations/ui";
import { Input } from "@canonical/react-components";
import React, { FC } from "react";
import { NodeInputProps } from "./helpers";

export const NodeInputText: FC<NodeInputProps> = ({
  attributes,
  node,
  setValue,
  disabled,
  dispatchSubmit,
  error,
}) => {
  return (
    <Input
      type="text"
      name={attributes.name}
      label={getNodeLabel(node)}
      disabled={disabled}
      defaultValue={node.messages.map(({ text }) => text).join(" ")}
      error={
        attributes.name === "code" ||
        attributes.name === "totp" ||
        attributes.name === "totp_code"
          ? error
          : undefined
      }
      onChange={(e) => void setValue(e.target.value)}
      onKeyDown={(e) => {
        if (e.key === "Enter") {
          e.preventDefault();
          e.stopPropagation();
          void dispatchSubmit(e, "password");
        }
      }}
    />
  );
};
