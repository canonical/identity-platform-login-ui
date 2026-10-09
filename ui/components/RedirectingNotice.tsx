import { Spinner } from "@canonical/react-components";
import React from "react";

interface Props {
  // The company sign-in the browser is being sent to.
  label: string;
}

export const RedirectingNotice = ({ label }: Props) => (
  <Spinner text={`Redirecting to ${label}…`} />
);
