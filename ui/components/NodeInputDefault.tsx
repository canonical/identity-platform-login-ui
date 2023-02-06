import { NodeInputProps } from "./helpers"

const TextInput = () => {
  return (
    <form>
      <label>
        <input type="text"/>
      </label>
      <input type="submit" value="Submit" />
    </form>
  );
}

export function NodeInputDefault(props: NodeInputProps) {
  // const { node, attributes, value = "", setValue, disabled } = props

  // Some attributes have dynamic JavaScript - this is for example required for WebAuthn.
  // const onClick = () => {
  //   // This section is only used for WebAuthn. The script is loaded via a <script> node
  //   // and the functions are available on the global window level. Unfortunately, there
  //   // is currently no better way than executing eval / function here at this moment.
  //   if (attributes.onclick) {
  //     const run = (() => attributes.onclick)
  //     run()
  //   }
  // }

  return <TextInput></TextInput>
}
