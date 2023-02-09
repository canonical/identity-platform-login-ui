import { getNodeLabel } from "@ory/integrations/ui"

import { NodeInputProps } from "./helpers"

const Checkbox = ({ label, value, onChange }) => {
  return (
    <label>
      <input type="checkbox" checked={value} onChange={onChange} />
      {label}
    </label>
  );
}

export function NodeInputCheckbox({
  node,
  attributes,
  setValue,
  disabled,
}: NodeInputProps) {
  // Render a checkbox.s
  return (
    <>
    <Checkbox
      label={getNodeLabel(node)}
      value={node.messages.map(({ text }) => text).join("\n")}
      onChange={(e) => setValue(e.target.checked)}
    />
      {/* <Checkbox
        name={attributes.name}
        defaultChecked={attributes.value === true}
        onChange={(e) => setValue(e.target.checked)}
        disabled={attributes.disabled || disabled}
        label={getNodeLabel(node)}
        state={
          node.messages.find(({ type }) => type === "error")
            ? "error"
            : undefined
        }
        subtitle={node.messages.map(({ text }) => text).join("\n")}
      /> */}
    </>
  )
}
