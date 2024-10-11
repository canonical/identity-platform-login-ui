import Image from "next/image";
import { UiNode, UiNodeImageAttributes } from "@ory/client";
import React, { FC } from "react";

interface Props {
  node: UiNode;
  attributes: UiNodeImageAttributes;
}

export const NodeImage: FC<Props> = ({ node, attributes }) => {
  return (
    <>
      {attributes.id === "totp_qr" && (
        <p>Open your authenticator app and scan the QR code</p>
      )}
      <Image
        data-testid={`node/image/${attributes.id}`}
        src={attributes.src}
        alt={node.meta.label?.text ?? ""}
      />
    </>
  );
};
