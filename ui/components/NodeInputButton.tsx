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
    if (attributes.onclick) {
      // handle passkeys that bring their own trigger
      e.stopPropagation();
      e.preventDefault();
      eval(attributes.onclick);
    } else {
      void setValue(attributes.value as string).then(() => dispatchSubmit(e));
    }
  };

  return (
    <>
      <Button
        onClick={handleClick}
        disabled={attributes.disabled || disabled}
        name={attributes.name}
      >
        {getNodeLabel(node)}
      </Button>
    </>
  );
};
