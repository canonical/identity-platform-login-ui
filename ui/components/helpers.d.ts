import { UiNode, UiNodeInputAttributes } from "@ory/client";
import { FormEvent } from "react";

export type ValueSetter = (
  value: string | number | boolean | undefined,
) => Promise<void>;

export type FormDispatcher = (
  e: MouseEvent | FormEvent,
  method?: string,
) => Promise<void>;

export interface NodeInputProps {
  node: UiNode;
  attributes: UiNodeInputAttributes;
  value: unknown;
  disabled: boolean;
  dispatchSubmit: FormDispatcher;
  setValue: ValueSetter;
  error?: string;
}
