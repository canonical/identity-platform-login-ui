import React from "react";
import { NextPage } from "next";
import PageLayout from "../components/PageLayout";
import { Col, Icon, Row } from "@canonical/react-components";
import { useRouter } from "next/router";

const secureAccount: NextPage = () => {
  const router = useRouter();
  return (
    <PageLayout title="Secure your account">
      <div
        className="p-card clickable"
        onClick={() => {
          void router.push("/setup_passkey");
        }}
      >
        <div className="p-card__content d-flex">
          <Row className="m-auto">
            <Col size={1} className="justify-items--end">
              <h1>
                <Icon name="private-key" />
              </h1>
            </Col>
            <Col size={5} className="d-flex">
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
          void router.push("/setup_secure");
        }}
      >
        <div className="p-card__content d-flex">
          <Row className="m-auto">
            <Col size={1} className="justify-items--end">
              <h1>
                <i className="p-icon--qr-code" />
              </h1>
            </Col>
            <Col size={5} className="d-flex">
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
