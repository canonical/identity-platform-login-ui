import { UiText } from "@ory/client";
import { Notification } from "@canonical/react-components";
import React from "react";
import { isErrorAnsweredByBackend } from "../util/constants";

type Severity = "negative" | "information" | "positive";

const toSeverity = (type: UiText["type"]): Severity => {
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
  types?: UiText["type"][];
}

// The flow-level messages a page shows: those of the given types, less the
// errors the backend answers in its own words.
export const visibleFlowMessages = (
  messages?: UiText[],
  types?: UiText["type"][],
): UiText[] =>
  (messages ?? []).filter(
    (message) =>
      (!types || types.includes(message.type)) &&
      !isErrorAnsweredByBackend(message),
  );

// Renders the flow-level messages that are not attached to a node, e.g. a
// rejection by the identity provider when returning from single sign-on.
export const FlowMessages = ({ messages, types }: Props) => {
  const shown = visibleFlowMessages(messages, types);
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
