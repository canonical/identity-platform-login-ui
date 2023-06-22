import { Row, CodeSnippet } from "@canonical/react-components";
import { AxiosError } from "axios";
import { useRouter } from "next/router";
import type { NextPage } from "next";
import { useEffect, useState } from "react";
import { kratos } from "../api/kratos";
import React from "react";
import Head from "next/head";
import { Session } from "@ory/client/api";

const Home: NextPage = () => {
  const [session, setSession] = useState<Session>();
  const router = useRouter();

  useEffect(() => {
    kratos
      .toSession()
      .then(void setSession)
      .catch((err: AxiosError) => {
        switch (err.response?.status) {
          case 403:
            // This is a legacy error code thrown. See code 422 for
            // more details.
            return router.push("/login?aal=aal2");
          case 422:
            // This status code is returned when we are trying to
            // validate a session which has not yet completed
            // its second factor
            return router.push("/login?aal=aal2");
          case 401:
            // do nothing, the user is not logged in
            return;
        }

        // Something else happened!
        return Promise.reject(err);
      });
  }, [router]);

  return (
    <>
      <Head>
        <title>Session Information</title>
      </Head>
      <Row className="p-strip">
        <CodeSnippet
          blocks={[
            {
              title: "Session Information",
              wrapLines: true,
              code: session
                ? JSON.stringify(session, null, 2)
                : "No valid Session was found.\nPlease sign in to receive one.",
            },
          ]}
        />
      </Row>
    </>
  );
};

export default Home;
