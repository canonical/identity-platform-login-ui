import { UiNodeAnchorAttributes } from "@ory/client";
import { UiNode } from "@ory/client";
import { Button } from "@canonical/react-components";
import React, { FC } from "react";

interface Props {
  node: UiNode;
  attributes: UiNodeAnchorAttributes;
}

export const NodeAnchor: FC<Props> = ({ attributes }) => {
  return (
    <Button
      data-testid={`node/anchor/${attributes.id}`}
      onClick={(e) => {
        e.stopPropagation();
        e.preventDefault();
        window.location.href = attributes.href;
      }}
    >
      {attributes.title.text}
    </Button>
  );
};
