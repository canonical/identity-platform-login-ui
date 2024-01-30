import React from "react";
import { CodeSnippet } from "@canonical/react-components";
import type { NextPage } from "next";
import { useRouter } from "next/router";
import PageLayout from "../components/PageLayout";

const OIDCError: NextPage = () => {
  const router = useRouter();
  const { error, error_description } = router.query;

  return (
    <PageLayout title="Sign in failed">
      <CodeSnippet
        blocks={[
          {
            wrapLines: true,
            code:
              router.isReady && error ? (
                error_description
              ) : (
                <>An error occurred please try again later.</>
              ),
          },
        ]}
      />
    </PageLayout>
  );
};

export default OIDCError;
