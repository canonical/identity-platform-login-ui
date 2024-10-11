import { getNodeLabel } from "@ory/integrations/ui";
import { Input } from "@canonical/react-components";
import React, { FC } from "react";
import { NodeInputProps } from "./helpers";

export const NodeInputEmail: FC<NodeInputProps> = ({
  node,
  value,
  setValue,
  disabled,
  dispatchSubmit,
  error: upstreamError,
}) => {
  const [hasLocalValidation, setLocalValidation] = React.useState(false);

  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  const isInvalid = !emailRegex.test((value as string) ?? "");
  const localError = isInvalid ? "Incorrect email address" : undefined;

  const error = hasLocalValidation ? localError : upstreamError;

  return (
    <Input
      type="email"
      label={getNodeLabel(node)}
      disabled={disabled}
      defaultValue={node.messages.map(({ text }) => text).join(" ")}
      error={error}
      onChange={(e) => void setValue(e.target.value)}
      onBlur={() => setLocalValidation(true)}
      onKeyDown={(e) => {
        if (e.key === "Enter") {
          e.preventDefault();
          e.stopPropagation();
          void dispatchSubmit(e, "code");
        }
      }}
    />
  );
};
