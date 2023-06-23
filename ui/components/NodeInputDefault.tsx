import { NodeInputProps } from "./helpers";
import React, { FC } from "react";

export const NodeInputDefault: FC<NodeInputProps> = () => {
  return (
    <form>
      <label>
        <input type="text" />
      </label>
      <input type="submit" value="Submit" />
    </form>
  );
};
