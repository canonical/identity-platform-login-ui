import { Card, Col, Navigation, Row, Theme } from "@canonical/react-components";
import React, { FC, ReactNode } from "react";
import Head from "next/head";
import SelfServeNavigation from "./SelfServeNavigation";

interface Props {
  children?: ReactNode;
  title: string;
  user?: string;
  isSelfServe?: boolean;
}

const PageLayout: FC<Props> = ({ children, title, user, isSelfServe }) => {
  return (
    <>
      <Head>
        <link
          rel="icon"
          type="image/png"
          sizes="32x32"
          href="https://assets.ubuntu.com/v1/be7e4cc6-COF-favicon-32x32.png"
        />
        <link
          rel="icon"
          type="image/png"
          sizes="16x16"
          href="https://assets.ubuntu.com/v1/16c27f81-COF%20favicon-16x16.png"
        />
        <title>{title}</title>
      </Head>
      {isSelfServe ? (
        <div className="l-application" role="presentation">
          <SelfServeNavigation user={user} />
          <main className="l-main">
            <div className="p-panel">
              <div className="p-panel__content">
                <Row>
                  <Col size={6}>
                    <h1 className="p-heading--4">{title}</h1>
                    {children}
                  </Col>
                </Row>
              </div>
            </div>
          </main>
        </div>
      ) : (
        <Row className="p-strip page-row">
          <Col emptyLarge={4} size={6}>
            <Card className="u-no-padding page-card">
              <Navigation
                logo={{
                  src: "https://assets.ubuntu.com/v1/82818827-CoF_white.svg",
                  title: "Canonical",
                  url: `./login`,
                }}
                theme={Theme.DARK}
              />
              <div className="p-card__inner page-inner">
                <h1 className="p-heading--4">{title}</h1>
                <div>{children}</div>
              </div>
            </Card>
          </Col>
        </Row>
      )}
    </>
  );
};

export default PageLayout;
