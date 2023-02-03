import { getNodeLabel } from "@ory/integrations/ui"
import { Button } from "@canonical/react-components";

import { NodeInputProps } from "./helpers"

export function NodeInputSubmit({
  node,
  attributes,
  setValue,
  disabled,
  dispatchSubmit,
}: NodeInputProps) {
  return (
    <Button
      // name={attributes.name}
      onClick={(e) => {
        // On click, we set this value, and once set, dispatch the submission!
        setValue(attributes.value).then(() => dispatchSubmit(e))
      }}
      disabled={attributes.disabled || disabled}
      className="login-button"
    >
      {getNodeLabel(node)}
    </Button>
  )
}
