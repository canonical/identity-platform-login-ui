import { Button, CodeSnippet } from "@canonical/react-components";
import { UiNode, UiNodeTextAttributes } from "@ory/client";
import { UiText } from "@ory/client";
import React, { FC, useCallback } from "react";

interface Props {
  node: UiNode;
  attributes: UiNodeTextAttributes;
}

interface ContextSecrets {
  secrets: UiText[];
}

const Content: FC<Props> = ({ attributes }) => {
  const copySecrets = useCallback((secrets: string[]) => {
    const codes = secrets.join("\n");
    void navigator.clipboard.writeText(codes);
  }, []);

  switch (attributes.text.id) {
    case 1050015:
      // This text node contains lookup secrets. Let's make them a bit more beautiful!
      // eslint-disable-next-line no-case-declarations
      const secrets = (attributes.text.context as ContextSecrets).secrets.map(
        (text) => {
          return text.id === 1050014 ? "Used" : text.text;
        },
      );

      return (
        <div
          className="container-fluid"
          data-testid={`node/text/${attributes.id}/text`}
        >
          <div className="row">
            <div className="u-sv1 u-no-print">
              <Button
                type="button"
                className="u-no-margin--bottom"
                onClick={() => copySecrets(secrets)}
              >
                Copy
              </Button>
              <Button
                type="button"
                className="u-no-margin--bottom"
                onClick={print}
              >
                Print
              </Button>
            </div>
            <ol className="p-list--divided backup-codes">
              {secrets.map((item, k) => (
                <li className="p-list__item" key={k}>
                  {item}
                </li>
              ))}
            </ol>
          </div>
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
  const isTotpSetup = attributes.id === "totp_secret_key";

  return (
    <>
      {isTotpSetup && <hr />}
      <p data-testid={`node/text/${attributes.id}/label`}>
        {isTotpSetup ? (
          <>
            Or <strong>if you can not scan the QR code</strong>, use the
            provided one-time code
          </>
        ) : (
          node.meta.label?.text
        )}
      </p>
      <Content node={node} attributes={attributes} />
      {isTotpSetup && <hr />}
    </>
  );
};
