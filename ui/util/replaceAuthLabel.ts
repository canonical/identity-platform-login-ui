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
                hasBackupLink: flow.ui.nodes.some(
                  (item) => item.group === "lookup_secret",
                ),
              },
            },
          };
        }
        return node;
      }),
    },
  };
};
