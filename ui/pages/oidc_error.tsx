import React from "react";
import { Icon, Row } from "@canonical/react-components";
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
      <Row className="p-strip is-shallow u-align--center">
        <Icon name="warning-grey" className="error-icon" />
        <h1 className="p-heading--4 error-heading">Login failed</h1>
        <div>
          {router.isReady && error ? (
            error_description
          ) : (
            <>An error occurred please try again later.</>
          )}
        </div>
      </Row>
    </>
  );
};

export default OIDCError;
