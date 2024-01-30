import { FlowError } from "@ory/client";
import { CodeSnippet } from "@canonical/react-components";
import { AxiosError, AxiosResponse } from "axios";
import type { NextPage } from "next";
import { useRouter } from "next/router";
import { useEffect, useState } from "react";
import React from "react";
import { kratos } from "../api/kratos";
import { GenericError } from "@ory/client/api";
import PageLayout from "../components/PageLayout";

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
      .then((res: AxiosResponse<FlowError>) => setError(res.data))
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
    <PageLayout title="Sign in failed">
      <CodeSnippet
        blocks={[
          {
            wrapLines: true,
            code:
              router.isReady && error?.error ? (
                (error.error as GenericError).reason ? (
                  (error.error as GenericError).reason
                ) : (
                  JSON.stringify(error.error, null, 2)
                )
              ) : (
                <>An error occurred. Try signing in again.</>
              ),
          },
        ]}
      />
    </PageLayout>
  );
};

export default Error;
