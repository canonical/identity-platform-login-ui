import { Button } from "@canonical/react-components";
import React, { FC, useCallback } from "react";
import { useRouter } from "next/router";

type ResetEmailBackButtonProps = {
  /** Explicit tab index for the button; defaults to undefined (natural tab order). */
  tabIndex?: number;
};

/**
 * A simple back button that navigates to the previous browser history entry.
 * Intended for use inside Kratos node metadata (reset_email page) where the
 * component is injected into `meta.label.context.beforeComponent`.
 */
export const ResetEmailBackButton: FC<ResetEmailBackButtonProps> = ({
  tabIndex,
}) => {
  const onClick = useCallback(() => window.history.back(), []);

  return (
    <Button tabIndex={tabIndex ?? 3} type="button" onClick={onClick}>
      Back
    </Button>
  );
};

export const BackButton: FC = () => {
  return (
    <Button
      tabIndex={3}
      type="button"
      onClick={() => {
        void window.history.back();
      }}
    >
      Back
    </Button>
  );
};
