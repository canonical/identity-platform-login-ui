import { UiNodeInputAttributes } from "@ory/client"
import { NodeInputButton } from "./NodeInputButton"
import { NodeInputCheckbox } from "./NodeInputCheckbox"
import { NodeInputDefault } from "./NodeInputDefault"
import { NodeInputHidden } from "./NodeInputHidden"
import { NodeInputSubmit } from "./NodeInputSubmit"
import { NodeInputProps } from "./helpers"

export function NodeInputOIDC(props: NodeInputProps) {
  const { node, value = "", setValue, disabled, dispatchSubmit} = props;
  const attributes: UiNodeInputAttributes = props.attributes;
  const provider = attributes.value.split('_')[0];
  if (provider === "hydra") {
    return <></>
  }
  const label = provider.charAt(0).toUpperCase() + provider.slice(1);
  node.meta.label.text = label;
  var p = {
    node: node, attributes: attributes, value: value, setValue: setValue, disabled: disabled, dispatchSubmit: dispatchSubmit
  }
  return <NodeInputSubmit {...p} />
}

export function NodeInput(props: NodeInputProps) {
  const { attributes } = props

  switch (attributes.type) {
    case "hidden":
      // Render a hidden input field
      return <NodeInputHidden {...props} />
    case "checkbox":
      // Render a checkbox. We have one hidden element which is the real value (true/false), and one
      // display element which is the toggle value (true)!
      return <NodeInputCheckbox {...props} />
    case "button":
      // Render a button
      return <NodeInputButton {...props} />
    case "submit":
      const { node } = props
      if (node.group === "oidc") {
        return <NodeInputOIDC {...props} />
      }
      // Render the submit button
      return <NodeInputSubmit {...props} />
  }
  // Render a generic text input field.
  return <NodeInputDefault {...props} />
}
