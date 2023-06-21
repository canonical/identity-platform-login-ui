import { getNodeLabel } from "@ory/integrations/ui";
import { NodeInputProps } from "./helpers";
import React, { FC } from "react";
import { Input } from "@canonical/react-components";

export const NodeInputCheckbox: FC<NodeInputProps> = ({ node, setValue }) => {
  return (
    <Input
      type="checkbox"
      label={getNodeLabel(node)}
      value={node.messages.map(({ text }) => text).join("\n")}
      onChange={(e) => void setValue(e.target.checked)}
    />
  );
};
