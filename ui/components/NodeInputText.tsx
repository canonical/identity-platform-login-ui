import { getNodeId, getNodeLabel } from "@ory/integrations/ui";
import { Button, Input } from "@canonical/react-components";
import React, { FC, useCallback, useEffect, useMemo } from "react";
import { NodeInputProps } from "./helpers";
import { FlowContext } from "../context/FlowContext";
import {
  isResendVerificationCode,
  isVerificationCodeInput,
} from "../util/constants";
import { UiNodeInputAttributes } from "@ory/client/api";
import { ORY_ERR_ACCOUNT_NOT_FOUND_OR_NO_LOGIN_METHOD } from "../util/constants";

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

  const [isDirty, setIsDirty] = React.useState(false);

  const flow = React.useContext(FlowContext);
  const urlParams = new URLSearchParams(window.location.search);
  const isWebauthn = urlParams.get("webauthn") === "true";
  const isIdentifierFirstGroup = node.group === "identifier_first";
  const isEmailInput = node.meta?.label?.text?.toLowerCase?.() === "email";

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

  useEffect(() => {
    if (isVerificationCodeInput(node)) {
      setIsDirty(false);
    }
  }, [node.messages]);

  useEffect(() => {
    if (isVerificationCodeInput(node)) {
      return;
    }
    if (message) {
      setInputValue(message);
    }
  }, [message, setInputValue]);

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
    const currentValue = (typeof value === "string" && value) || "";
    if (
      isIdentifierFirstGroup &&
      isEmailInput &&
      attributes.value === currentValue
    ) {
      const serverErrorNode = node.messages.find((m) => m.type === "error");

      if (!serverErrorNode) return undefined;

      const serverErrorId = serverErrorNode?.id;
      const serverErrorText = serverErrorNode?.text ?? "";

      if (currentValue.length === 0) {
        return "Please enter your email address.";
      }

      if (!/^\S+@\S+\.\S+$/.test(currentValue)) {
        return "Enter a valid email address.";
      }

      if (serverErrorId === ORY_ERR_ACCOUNT_NOT_FOUND_OR_NO_LOGIN_METHOD) {
        return "No account found for this email. Verify or create a new account.";
      }

      if (serverErrorText) {
        return serverErrorText;
      }
    }

    if (node && isVerificationCodeInput(node)) {
      if (isDirty) {
        return;
      }
    }

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
  }, [
    message,
    node.messages,
    isDuplicate,
    attributes.name,
    isWebauthn,
    error,
    isDirty,
  ]);

  const getPlaceholderText = () => {
    if (isIdentifierFirstGroup && isEmailInput) {
      return "Your Email";
    }
    return "";
  };

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
              (resendButton?.attributes as UiNodeInputAttributes)?.disabled ||
              false
            }
          >
            Resend code
          </Button>
        </>
      );
    }
    return getNodeLabel(node);
  }, [node, flow]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (e.key === "Enter") {
        e.preventDefault();
        e.stopPropagation();
        setIsDirty(false);
        void dispatchSubmit(e, isIdentifierFirstGroup ? "identifier_first" : "password");
      }
    },
    [dispatchSubmit],
  );
  const handleChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      let newValue = e.target.value;
      if (isVerificationCodeInput(node)) {
        setIsDirty(true);
        newValue = newValue.replace(/[^0-9]/g, "");
      }
      setInputValue(newValue);
      void setValue(newValue);
    },
    [setValue],
  );
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
        placeholder={getPlaceholderText()}
        value={inputValue}
        error={getError}
        onChange={handleChange}
        onKeyDown={handleKeyDown}
        maxLength={isVerificationCodeInput(node) ? 6 : undefined}
      />
      {afterComponent}
    </>
  );
};
