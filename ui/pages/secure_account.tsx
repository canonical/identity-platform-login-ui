import React from "react";
import { NextPage } from "next";
import PageLayout from "../components/PageLayout";
import { Col, Icon, Row } from "@canonical/react-components";
import { useRouter } from "next/router";
import { redirectTo } from "../util/redirectTo";

const secureAccount: NextPage = () => {
  const router = useRouter();
  return (
    <PageLayout title="Secure your account">
      <div
        className="p-card clickable"
        onClick={() => {
          redirectTo("http://localhost/ui/setup_passkey", router);
        }}
      >
        <div className="p-card__content d-flex">
          <Row className="m-auto">
            <Col size={1} medium={1} small={1} className="justify-items--end">
              <h1 className="p-heading--display">
                <Icon name="private-key" />
              </h1>
            </Col>
            <Col size={5} medium={5} small={3} className="d-flex">
              <p style={{ marginBlock: "auto" }}>
                <strong>Set up Passkey (recommended)</strong> <br />
                <small>
                  Verify with your FaceID, TouchID, PIN or Security key.
                </small>
              </p>
            </Col>
          </Row>
        </div>
      </div>
      <div
        className="p-card clickable"
        onClick={() => {
          redirectTo("http://localhost/ui/setup_secure", router);
        }}
      >
        <div className="p-card__content d-flex">
          <Row className="m-auto">
            <Col size={1} medium={1} small={1} className="justify-items--end">
              <h1 className="p-heading--display">
                <i className="p-icon--qr-code" />
              </h1>
            </Col>
            <Col size={5} medium={5} small={3} className="d-flex">
              <p style={{ marginBlock: "auto" }}>
                <strong>Set up an Authenticator App</strong> <br />
                <small>
                  Verify by entering a 6-digit code from your authenticator app.
                </small>
              </p>
            </Col>
          </Row>
        </div>
      </div>
    </PageLayout>
  );
};

export default secureAccount;
