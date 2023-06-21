import { CodeSnippet } from "@canonical/react-components";
import { UiNode, UiNodeTextAttributes } from "@ory/client";
import { UiText } from "@ory/client";
import React, { FC } from "react";

interface Props {
  node: UiNode;
  attributes: UiNodeTextAttributes;
}

interface ContextSecrets {
  secrets: UiText[];
}

const Content: FC<Props> = ({ attributes }) => {
  switch (attributes.text.id) {
    case 1050015:
      // This text node contains lookup secrets. Let's make them a bit more beautiful!
      // eslint-disable-next-line no-case-declarations
      const secrets = (attributes.text.context as ContextSecrets).secrets.map(
        (text, k) => (
          <div
            key={k}
            data-testid={`node/text/${attributes.id}/lookup_secret`}
            className="col-xs-3"
          >
            {/* Used lookup_secret has ID 1050014 */}
            <code>{text.id === 1050014 ? "Used" : text.text}</code>
          </div>
        )
      );
      return (
        <div
          className="container-fluid"
          data-testid={`node/text/${attributes.id}/text`}
        >
          <div className="row">{secrets}</div>
        </div>
      );
  }

  return (
    <div data-testid={`node/text/${attributes.id}/text`}>
      <CodeSnippet
        blocks={[
          {
            wrapLines: true,
            code: attributes.text.text,
          },
        ]}
      />
    </div>
  );
};

export const NodeText: FC<Props> = ({ node, attributes }) => {
  return (
    <>
      <p data-testid={`node/text/${attributes.id}/label`}>
        {node.meta.label?.text}
      </p>
      <Content node={node} attributes={attributes} />
    </>
  );
};
