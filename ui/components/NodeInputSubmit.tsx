import { getNodeLabel } from "@ory/integrations/ui";
import { Button } from "@canonical/react-components";
import { NodeInputProps } from "./helpers";
import React, { FC } from "react";

export const NodeInputSubmit: FC<NodeInputProps> = ({
  node,
  attributes,
  setValue,
  disabled,
  dispatchSubmit,
}) => (
  <Button
    onClick={async (e) => {
      // On click, we set this value, and once set, dispatch the submission!
      await setValue(attributes.value as string).then(() => dispatchSubmit(e));
    }}
    disabled={attributes.disabled || disabled}
    className="login-button"
  >
    {getNodeLabel(node)}
  </Button>
);
