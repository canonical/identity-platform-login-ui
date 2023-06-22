import React from "react";
import { Notification, Row } from "@canonical/react-components";
import type { NextPage } from "next";
import { useRouter } from "next/router";
import Head from "next/head";

const OIDCError: NextPage = () => {
  const router = useRouter();
  const { error, error_description } = router.query;

  return (
    <>
      <Head>
        <title>Login failed</title>
      </Head>
      <Row className="p-strip">
        <h1 className="p-heading--3">Login failed</h1>
        {router.isReady && error ? (
          <Notification title={error} severity="negative">
            {error_description}
          </Notification>
        ) : (
          <Notification title="An error occurred" severity="negative" />
        )}
      </Row>
    </>
  );
};

export default OIDCError;
