import React from "react";
import {
  isUiNodeAnchorAttributes,
  isUiNodeImageAttributes,
  isUiNodeInputAttributes,
  isUiNodeScriptAttributes,
  isUiNodeTextAttributes,
} from "@ory/integrations/ui";
import { NodeAnchor } from "./NodeAnchor";
import { NodeImage } from "./NodeImage";
import { NodeInput } from "./NodeInput";
import { NodeScript } from "./NodeScript";
import { NodeText } from "./NodeText";
import { NodeInputProps } from "./helpers";
import { FC } from "react";

export const Node: FC<Omit<NodeInputProps, "attributes">> = ({
  node,
  value,
  setValue,
  disabled,
  dispatchSubmit,
  error,
}) => {
  if (isUiNodeImageAttributes(node.attributes)) {
    return <NodeImage node={node} attributes={node.attributes} />;
  }

  if (isUiNodeScriptAttributes(node.attributes)) {
    return <NodeScript node={node} attributes={node.attributes} />;
  }

  if (isUiNodeTextAttributes(node.attributes)) {
    return <NodeText node={node} attributes={node.attributes} />;
  }

  if (isUiNodeAnchorAttributes(node.attributes)) {
    return <NodeAnchor node={node} attributes={node.attributes} />;
  }

  if (isUiNodeInputAttributes(node.attributes)) {
    return (
      <NodeInput
        dispatchSubmit={dispatchSubmit}
        value={value as string}
        setValue={setValue}
        node={node}
        disabled={disabled}
        attributes={node.attributes}
        error={error}
      />
    );
  }

  return null;
};
