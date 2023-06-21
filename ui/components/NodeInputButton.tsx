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
  // Some attributes have dynamic JavaScript - this is for example required for WebAuthn.
  const handleClick = (e: MouseEvent | FormEvent) => {
    // This section is only used for WebAuthn. The script is loaded via a <script> node
    // and the functions are available on the global window level. Unfortunately, there
    // is currently no better way than executing eval / function here at this moment.
    //
    // Please note that we also need to prevent the default action from happening.
    if (attributes.onclick) {
      e.stopPropagation();
      e.preventDefault();
      const run = () => attributes.onclick; // todo wtf?
      run();
      return;
    }

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
