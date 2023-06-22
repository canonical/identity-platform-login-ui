import { FlowError } from "@ory/client";
import { Col, Notification, Row } from "@canonical/react-components";
import { AxiosError } from "axios";
import type { NextPage } from "next";
import { useRouter } from "next/router";
import { useEffect, useState } from "react";
import React from "react";
import { kratos } from "../api/kratos";
import Head from "next/head";

const Error: NextPage = () => {
  const [error, setError] = useState<FlowError>();

  // Get ?id=... from the URL
  const router = useRouter();
  const { id } = router.query;

  useEffect(() => {
    // If the router is not ready yet, or we already have an error, do nothing.
    if (!router.isReady || error) {
      return;
    }

    kratos
      .getFlowError({ id: String(id) })
      .then(({ data }) => {
        setError(data);
      })
      .catch((err: AxiosError) => {
        switch (err.response?.status) {
          case 404:
            // The error id could not be found. Let's just redirect home!
            return router.push("/");
          case 403:
            // The error id could not be fetched due to e.g. a CSRF issue. Let's just redirect home!
            return router.push("/");
          case 410:
            // The error id expired. Let's just redirect home!
            return router.push("/");
        }

        return Promise.reject(err);
      });
  }, [id, router, router.isReady, error]);

  return (
    <>
      <Head>
        <title>Login failed</title>
      </Head>
      <Row className="p-strip">
        <Col size={12}>
          <h1 className="p-heading--3">Login failed</h1>
          {router.isReady && error?.error ? (
            <Notification title="An error occurred" severity="negative">
              {JSON.stringify(error.error, null, 2)}
            </Notification>
          ) : (
            <Notification title="An error occurred" severity="negative" />
          )}
        </Col>
      </Row>
    </>
  );
};

export default Error;
