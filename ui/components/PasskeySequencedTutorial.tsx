import type { NextPage } from "next";
import React from "react";
import { Accordion } from "@canonical/react-components";

const PasskeySequencedTutorial: NextPage = () => {
  return (
    <>
      <p className="u-text--muted u-sv-3">
        Authentication setup needed to continue.
      </p>
      <h2 className="p-heading--4">Add a security key</h2>
      <Accordion
        sections={[
          {
            title: "How to add a Security key",
            content: (
              <>
                <ol className="p-list--nested-counter">
                  <li>
                    Enter a name for your security key (like {'"'}iPhone
                    {'"'} or {'"'}Work Laptop{'"'})
                  </li>
                  <li>
                    Click {'"'}Add security key{'"'}
                  </li>

                  <li>
                    When your browser prompts you, use your device{"'"}s
                    fingerprint, face recognition, or PIN to verify
                  </li>
                  <li>Wait for confirmation</li>
                </ol>
                <p>
                  That{"'"}s it! Your security key is now set up for future
                  sign-ins.
                </p>
                <p>
                  (Note: Your device needs biometrics or a PIN already
                  configured)
                </p>
              </>
            ),
          },
        ]}
      />
    </>
  );
};

export default PasskeySequencedTutorial;
