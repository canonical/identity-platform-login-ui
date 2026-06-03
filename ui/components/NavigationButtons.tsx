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

type FlowBackButtonProps<T> = {
  setFlow?: React.Dispatch<React.SetStateAction<T | undefined>>;
  type?: "submit" | "reset" | "button";
  appearance?: string;
  disabled?: boolean;
  className?: string;
  tabIndex?: number;
  text?: string;
};

export const FlowBackButton = <T,>({
  setFlow,
  ...props
}: FlowBackButtonProps<T>) => {
  const router = useRouter();

  const onBack = useCallback(async () => {
    const { flow: _, ...noFlowQueryParams } = router.query;
    void _; // ignoring the flow id to "restart" the flow

    try {
      await router.replace({
        query: noFlowQueryParams,
      });
    } catch (e) {
      console.error("Error replacing route:", e);
      throw e;
    }

    if (setFlow) setFlow(undefined);
  }, [router, setFlow]);

  return (
    <Button
      tabIndex={props.tabIndex}
      type={props.type ?? "button"}
      onClick={onBack}
      disabled={props.disabled ?? false}
      className={props.className ?? ""}
      appearance={props.appearance ?? "secondary"}
    >
      {props.text ?? "Back"}
    </Button>
  );
};
