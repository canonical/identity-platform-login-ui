import Image from "next/image"
import { UiNode, UiNodeImageAttributes } from "@ory/client"

interface Props {
  node: UiNode
  attributes: UiNodeImageAttributes
}

export const NodeImage = ({ node, attributes }: Props) => {
  return (
    <Image
      data-testid={`node/image/${attributes.id}`}
      src={attributes.src}
      alt={node.meta.label?.text}
    />
  )
}
