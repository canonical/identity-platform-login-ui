import { getNodeLabel } from "@ory/integrations/ui";
import { Input } from "@canonical/react-components";
import React, { FC } from "react";
import { NodeInputProps } from "./helpers";

export const NodeInputText: FC<NodeInputProps> = ({
  attributes,
  node,
  value,
  setValue,
  disabled,
  dispatchSubmit,
  error,
}) => {
  const urlParams = new URLSearchParams(window.location.search);
  const isWebauthn = urlParams.get("webauthn") === "true";
  const ucFirst = (s?: string) =>
    s ? String(s[0]).toUpperCase() + String(s).slice(1) : s;

  let deduplicateValues: string[] = [];
  if (node.meta.label?.context) {
    if ("deduplicateValues" in node.meta.label.context) {
      deduplicateValues = node.meta.label.context.deduplicateValues as string[];
    }
  }
  const isDuplicate = deduplicateValues.includes(value as string);

  const message = node.messages.map(({ text }) => text).join(" ");
  const defaultValue = message.includes("Invalid login method")
    ? (value as string)
    : message;

  const getError = () => {
    if (message.startsWith("Invalid login method")) {
      return "Invalid login method";
    }

    if (isDuplicate) {
      return "This value is already in use";
    }

    if (
      attributes.name === "code" ||
      attributes.name === "totp" ||
      attributes.name === "totp_code" ||
      attributes.name === "lookup_secret" ||
      (isWebauthn && attributes.name === "identifier")
    ) {
      return ucFirst(error);
    }

    return undefined;
  };

  return (
    <Input
      type="text"
      autoFocus={true}
      tabIndex={1}
      name={attributes.name}
      label={getNodeLabel(node)}
      disabled={disabled}
      defaultValue={defaultValue}
      error={getError()}
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
