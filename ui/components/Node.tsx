import React, { FC } from "react";
import {
  isUiNodeAnchorAttributes,
  isUiNodeImageAttributes,
  isUiNodeInputAttributes,
  isUiNodeTextAttributes,
} from "@ory/integrations/ui";
import { NodeAnchor } from "./NodeAnchor";
import { NodeImage } from "./NodeImage";
import { NodeInput } from "./NodeInput";
import { NodeText } from "./NodeText";
import { NodeInputProps } from "./helpers";
import { FlowBackButton } from "./NavigationButtons";
import { UiNodeInputAttributes } from "@ory/client";
import { isUiNodeBackButton } from "../util/constants";

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

  if (isUiNodeTextAttributes(node.attributes)) {
    return <NodeText node={node} attributes={node.attributes} />;
  }

  if (isUiNodeAnchorAttributes(node.attributes)) {
    return <NodeAnchor node={node} attributes={node.attributes} />;
  }

  if (isUiNodeBackButton(node.meta)) {
    const attrs = node.attributes as UiNodeInputAttributes;
    return (
      <FlowBackButton disabled={attrs.disabled} text={node.meta.label?.text} />
    );
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
