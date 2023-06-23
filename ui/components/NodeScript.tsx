import { UiNode, UiNodeScriptAttributes } from "@ory/client";
import { FC, useEffect } from "react";

interface Props {
  node: UiNode;
  attributes: UiNodeScriptAttributes;
}

export const NodeScript: FC<Props> = ({ attributes }) => {
  useEffect(() => {
    const script = document.createElement("script");

    script.async = true;
    script.setAttribute("data-testid", `node/script/${attributes.id}`);
    script.src = attributes.src;
    script.async = attributes.async;
    script.crossOrigin = attributes.crossorigin;
    script.integrity = attributes.integrity;
    script.referrerPolicy = attributes.referrerpolicy;
    script.type = attributes.type;

    document.body.appendChild(script);

    return () => {
      document.body.removeChild(script);
    };
  }, [attributes]);

  return null;
};
