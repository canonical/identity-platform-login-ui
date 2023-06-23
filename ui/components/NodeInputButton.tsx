import { getNodeLabel } from "@ory/integrations/ui";
import { Button } from "@canonical/react-components";
import React, { FC, FormEvent } from "react";
import { NodeInputProps } from "./helpers";

export const NodeInputButton: FC<NodeInputProps> = ({
  node,
  attributes,
  setValue,
  disabled,
  dispatchSubmit,
}) => {
  const handleClick = (e: MouseEvent | FormEvent) => {
    void setValue(attributes.value as string).then(() => dispatchSubmit(e));
  };

  return (
    <>
      <Button onClick={handleClick} disabled={attributes.disabled || disabled}>
        {getNodeLabel(node)}
      </Button>
    </>
  );
};
