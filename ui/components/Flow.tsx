import {
  LoginFlow,
  RecoveryFlow,
  UiNode,
  UpdateLoginFlowBody,
  UpdateRecoveryFlowBody,
  UpdateRegistrationFlowBody,
  UpdateSettingsFlowBody,
  UpdateVerificationFlowBody,
} from "@ory/client";
import { getNodeId, isUiNodeInputAttributes } from "@ory/integrations/ui";
import React, { Component, FormEvent } from "react";
import { Node } from "./Node";

export type Values = Partial<
  | UpdateLoginFlowBody
  | UpdateRegistrationFlowBody
  | UpdateRecoveryFlowBody
  | UpdateSettingsFlowBody
  | UpdateVerificationFlowBody
>;

export type Methods = "oidc" | "password" | "code" | "link";

export interface Props<T> {
  // The flow
  flow?: LoginFlow | RecoveryFlow;
  // Only show certain nodes. We will always render the default nodes for CSRF tokens.
  only?: Methods;
  // Is triggered on submission
  onSubmit: (values: T) => Promise<void>;
  // Do not show the global messages. Useful when rendering them elsewhere.
  hideGlobalMessages?: boolean;
}

interface State<T> {
  values: T;
  isLoading: boolean;
  error?: string;
}

export class Flow<T extends Values> extends Component<Props<T>, State<T>> {
  constructor(props: Props<T>) {
    super(props);
    this.state = {
      values: {} as T,
      isLoading: false,
    };
  }

  componentDidMount() {
    this.initializeValues(this.filterNodes());
  }

  componentDidUpdate(prevProps: Props<T>) {
    if (prevProps.flow !== this.props.flow) {
      // Flow has changed, reload the values!
      this.initializeValues(this.filterNodes());
    }
  }

  initializeValues = (nodes: UiNode[] = []) => {
    // Compute the values
    const values = {} as T;
    nodes.forEach((node) => {
      // This only makes sense for text nodes
      if (isUiNodeInputAttributes(node.attributes)) {
        if (
          node.attributes.type === "button" ||
          node.attributes.type === "submit"
        ) {
          // In order to mimic real HTML forms, we need to skip setting the value
          // for buttons as the button value will (in normal HTML forms) only trigger
          // if the user clicks it.
          return;
        }
        const key = node.attributes.name as keyof Values;
        values[key] = node.attributes.value as Values[keyof Values];
      }
    });

    // Set all the values!
    this.setState((state) => ({ ...state, values }));
  };

  filterNodes = (): UiNode[] => {
    const { flow, only } = this.props;
    if (!flow || !flow.ui.nodes) {
      return [];
    }
    return flow.ui.nodes.filter(({ group }) => {
      if (!only) {
        return true;
      }
      return group === "default" || group === only;
    });
  };

  // Handles form submission
  handleSubmit = (e: MouseEvent | FormEvent, method?: string) => {
    // Prevent all native handlers
    e.stopPropagation();
    e.preventDefault();

    // Prevent double submission!
    if (this.state.isLoading) {
      return Promise.resolve();
    }

    this.setState((state) => {
      return {
        ...state,
        isLoading: true,
      };
    });

    const values = method
      ? { ...this.state.values, method }
      : this.state.values;

    return this.props
      .onSubmit(values)
      .catch((e) => {
        const backendError = e as { response?: { data?: string } };
        this.setState((state) => ({
          ...state,
          error: backendError?.response?.data?.toString(),
        }));
      })
      .finally(() => {
        // We wait for reconciliation and update the state after 50ms
        // Done submitting - update loading status
        this.setState((state) => ({
          ...state,
          isLoading: false,
        }));
      });
  };

  render() {
    const { flow } = this.props;
    const { values, isLoading, error } = this.state;

    // Filter the nodes - only show the ones we want
    const nodes = this.filterNodes();

    if (!flow) {
      // No flow was set yet? It's probably still loading...
      //
      // Nodes have only one element? It is probably just the CSRF Token
      // and the filter did not match any elements!
      return null;
    }

    return (
      <form
        action={flow.ui.action}
        method={flow.ui.method}
        onSubmit={void this.handleSubmit}
        className={error && "is-error"}
      >
        {nodes.map((node, k) => {
          const id = getNodeId(node) as keyof Values;
          return (
            <Node
              key={`${id}-${k}`}
              disabled={isLoading}
              error={error}
              node={node}
              value={values[id]}
              dispatchSubmit={this.handleSubmit}
              setValue={(value) => {
                return new Promise((resolve) => {
                  this.setState(
                    (state) => ({
                      ...state,
                      values: {
                        ...state.values,
                        [getNodeId(node)]: value,
                      },
                    }),
                    resolve,
                  );
                });
              }}
            />
          );
        })}
      </form>
    );
  }
}
