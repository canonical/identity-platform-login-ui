import { UiText } from "@ory/client";
import { Notification } from "@canonical/react-components";
import React from "react";
import { isErrorAnsweredByBackend } from "../util/constants";

type Severity = "negative" | "information" | "positive";

const toSeverity = (type: string): Severity => {
  if (type === "error") {
    return "negative";
  }
  if (type === "success") {
    return "positive";
  }
  return "information";
};

interface Props {
  // Flow-level messages, i.e. flow.ui.messages
  messages?: UiText[];
  // Only render messages of these types, all of them when omitted
  types?: string[];
}

// Renders the flow-level messages that are not attached to a node, e.g. a
// rejection by the identity provider when returning from single sign-on.
export const FlowMessages = ({ messages, types }: Props) => {
  const shown = (messages ?? []).filter(
    (message) =>
      (!types || types.includes(message.type)) &&
      !isErrorAnsweredByBackend(message),
  );
  if (shown.length === 0) {
    return null;
  }

  return (
    <>
      {shown.map(({ id, type, text }, k) => (
        <Notification
          key={`${id}-${k}`}
          severity={toSeverity(type)}
          inline
          role={type === "error" ? "alert" : "status"}
        >
          {text}
        </Notification>
      ))}
    </>
  );
};
