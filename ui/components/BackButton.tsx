import { Button } from "@canonical/react-components";
import React, { FC } from "react";

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
