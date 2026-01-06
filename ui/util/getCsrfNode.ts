import { UiNode, UiNodeInputAttributes } from "@ory/client";

export const getCsrfNode = (
  nodes?: UiNode[],
): UiNode | undefined =>
  nodes?.find(
    (node) =>
      node.group === "default" &&
      // ensure it's an input node with the csrf token name
      (node.attributes as UiNodeInputAttributes | undefined)?.node_type ===
        "input" &&
      (node.attributes as UiNodeInputAttributes | undefined)?.name ===
        "csrf_token",
  );

export const getCsrfToken = (nodes?: UiNode[]): string | undefined => {
  const node = getCsrfNode(nodes);
  return (node?.attributes as UiNodeInputAttributes | undefined)?.value as
    | string
    | undefined;
};
