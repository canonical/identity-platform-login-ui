import { NodeInputProps } from "./helpers";
import React, { FC } from "react";
import { Input } from "@canonical/react-components";

export const NodeInputHidden: FC<NodeInputProps> = ({
  node,
  attributes,
  setValue,
}) => {
  if (node.meta.label?.text === "Change e-mail") {
    return (
      <Input
        type="email"
        labelClassName="password-label"
        label={
          <>
            <span>E-Mail</span>
            <a
              href={`${window.location.href}&reset_email=1`}
              style={{ float: "right" }}
            >
              Change e-mail
            </a>
          </>
        }
        onChange={(e) => void setValue(e.target.value)}
        disabled={true}
        defaultValue={attributes.value as string}
      />
    );
  }

  return (
    <input
      type={attributes.type}
      name={attributes.name}
      value={(attributes.value as string) || "true"}
    />
  );
};
