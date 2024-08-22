import { Button, CodeSnippet, List } from "@canonical/react-components";
import { UiNode, UiNodeTextAttributes } from "@ory/client";
import { UiText } from "@ory/client";
import React, { FC } from "react";
import ReactPDF from "@react-pdf/renderer";
import BackupCodePdf from "./BackupCodePdf";

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
            <List
              items={secrets.map((item, k) => (
                <span key={k}>{item}</span>
              ))}
              divided
            />
            <div className="u-no-print">
              <Button
                type="button"
                onClick={async () => {
                  const blob = await ReactPDF.pdf(
                    <BackupCodePdf codes={secrets} />,
                  ).toBlob();

                  const link = document.createElement("a");
                  link.href = URL.createObjectURL(blob);
                  link.download = "backup-codes.pdf";
                  link.click();

                  // Clean up the URL object
                  URL.revokeObjectURL(link.href);
                }}
              >
                Download
              </Button>
              <Button
                type="button"
                onClick={() => {
                  const codes = secrets.join("\n");
                  void navigator.clipboard.writeText(codes);
                }}
              >
                Copy
              </Button>
              <Button type="button" onClick={print}>
                Print
              </Button>
            </div>
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
  return (
    <>
      <p data-testid={`node/text/${attributes.id}/label`}>
        {node.meta.label?.text}
      </p>
      <Content node={node} attributes={attributes} />
    </>
  );
};
