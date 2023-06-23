import { NodeInputProps } from "./helpers";
import React, { FC } from "react";

export const NodeInputHidden: FC<NodeInputProps> = ({ attributes }) => {
  return (
    <input
      type={attributes.type}
      name={attributes.name}
      value={(attributes.value as string) || "true"}
    />
  );
};
