import { LoginFlow } from "@ory/client";

export const replaceAuthLabel = (
  flow: LoginFlow | undefined,
): LoginFlow | undefined => {
  if (!flow) {
    return flow;
  }
  return {
    ...flow,
    ui: {
      ...flow?.ui,
      nodes: flow?.ui.nodes.map((node) => {
        if (node.meta.label?.text === "Use Authenticator") {
          return {
            ...node,
            meta: {
              ...node.meta,
              label: {
                ...node.meta.label,
                text: "Sign in",
              },
            },
          };
        }
        return node;
      }),
    },
  };
};
