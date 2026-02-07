import { getNodeId, getNodeLabel } from "@ory/integrations/ui";
import { Button, Input } from "@canonical/react-components";
import React, { FC, useEffect, useMemo } from "react";
import { NodeInputProps } from "./helpers";
import { FlowContext } from "../context/FlowContext";
import {
  isResendVerificationCode,
  isVerificationCodeInput,
} from "../util/constants";
import { UiNodeInputAttributes } from "@ory/client/api";

export const NodeInputText: FC<NodeInputProps> = ({
  attributes,
  node,
  value,
  setValue,
  disabled,
  dispatchSubmit,
  error,
}) => {
  const [inputValue, setInputValue] = React.useState(
    attributes.value as string,
  );

  const flow = React.useContext(FlowContext);
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

  const message = useMemo(
    () => node.messages.map(({ text }) => text).join(" "),
    [node.messages],
  );

  const beforeComponent = (
    node.meta.label?.context as {
      beforeComponent: React.ReactNode;
    }
  )?.beforeComponent;

  const afterComponent = (
    node.meta.label?.context as {
      afterComponent: React.ReactNode;
    }
  )?.afterComponent;

  useEffect(() => {
    if (node.messages.length === 0) {
      return;
    }
    for (const msg of node.messages) {
      if (msg.type !== "info") {
        return;
      }
    }
    if (message) {
      setInputValue(message);
    }
  }, [message, setInputValue]);

  const getError = useMemo(() => {
    if (message.startsWith("Invalid login method")) {
      return "Invalid login method";
    }

    if (node.messages.length > 0) {
      return node.messages
        .filter((msg) => msg.type === "error")
        .map((msg) => msg.text)
        .join(" ");
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
  }, [message, node.messages, isDuplicate, attributes.name, isWebauthn, error]);

  const getLabel = useMemo(() => {
    if (isVerificationCodeInput(node)) {
      const resendButton = flow.ui.nodes.find(isResendVerificationCode);
      return (
        <>
          <span>{getNodeLabel(node)}</span>
          <Button
            appearance={"link"}
            tabIndex={4}
            onClick={async (e) => {
              e.preventDefault();
              // On click, we set this value, and once set, dispatch the submission!
              await setValue(
                (resendButton?.attributes as UiNodeInputAttributes)
                  .value as string,
                resendButton ? getNodeId(resendButton) : undefined,
              ).then(() => dispatchSubmit(e));
            }}
            style={{ float: "right", marginBottom: 0 }}
            disabled={
              (resendButton?.attributes as UiNodeInputAttributes).disabled
            }
          >
            Resend code
          </Button>
        </>
      );
    }
    return getNodeLabel(node);
  }, [node, flow]);

  return (
    <>
      {beforeComponent}
      <Input
        type="text"
        autoFocus={true}
        tabIndex={1}
        name={attributes.name}
        label={getLabel}
        labelClassName="password-label"
        disabled={disabled}
        value={inputValue}
        error={getError}
        onChange={(e) => {
          const newValue = e.target.value;
          setInputValue(newValue);
          void setValue(newValue);
        }}
        onKeyDown={(e) => {
          if (e.key === "Enter") {
            e.preventDefault();
            e.stopPropagation();
            void dispatchSubmit(e, "password");
          }
        }}
        maxLength={isVerificationCodeInput(node) ? 6 : undefined}
      />
      {afterComponent}
    </>
  );
};
