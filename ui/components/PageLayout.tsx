import { Card, Col, Navigation, Row, Theme } from "@canonical/react-components";
import React, { FC, ReactNode, useLayoutEffect } from "react";
import Head from "next/head";

interface Props {
  children?: ReactNode;
  title: string;
}

const PageLayout: FC<Props> = ({ children, title }) => {
  useLayoutEffect(() => {
    document.querySelector("body")?.classList.add("is-paper");
  });

  return (
    <>
      <Head>
        <title>{title}</title>
      </Head>
      <Row className="p-strip page-row">
        <Col emptyLarge={4} size={6}>
          <Card className="u-no-padding page-card">
            <Navigation
              logo={{
                src: "https://assets.ubuntu.com/v1/82818827-CoF_white.svg",
                title: "Canonical",
                url: `/`,
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
    </>
  );
};

export default PageLayout;
