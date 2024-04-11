import { NodeInputButton } from "./NodeInputButton";
import { NodeInputCheckbox } from "./NodeInputCheckbox";
import { NodeInputDefault } from "./NodeInputDefault";
import { NodeInputHidden } from "./NodeInputHidden";
import { NodeInputSubmit } from "./NodeInputSubmit";
import { NodeInputProps } from "./helpers";
import React, { FC } from "react";
import { NodeInputText } from "./NodeInputText";
import { NodeInputPassword } from "./NodeInputPassword";

export const NodeInputOIDC: FC<NodeInputProps> = ({
  attributes,
  node,
  value = "",
  setValue,
  disabled,
  dispatchSubmit,
}) => {
  if (node.meta.label?.context) {
    if ("provider" in node.meta.label.context) {
      const provider = String(node.meta.label.context.provider);
      node.meta.label.text =
        provider.charAt(0).toUpperCase() + provider.slice(1);
    }
  }
  const props = {
    node: node,
    attributes: attributes,
    value: value as string,
    setValue: setValue,
    disabled: disabled,
    dispatchSubmit: dispatchSubmit,
  };
  return <NodeInputSubmit {...props} />;
};

export const NodeInput: FC<NodeInputProps> = (props) => {
  const { attributes } = props;

  switch (attributes.type) {
    case "hidden":
      return <NodeInputHidden {...props} />;
    case "checkbox":
      // Render a checkbox. We have one hidden element which is the real value (true/false), and one
      // display element which is the toggle value (true)!
      return <NodeInputCheckbox {...props} />;
    case "button":
      return <NodeInputButton {...props} />;
    case "text":
      return <NodeInputText {...props} />;
    case "password":
      return <NodeInputPassword {...props} />;
    case "submit":
      if (props.node.group === "oidc") {
        return <NodeInputOIDC {...props} />;
      }
      return <NodeInputSubmit {...props} />;
  }
  // Render a generic text input field.
  return <NodeInputDefault {...props} />;
};
