import type { NextPage } from "next";
import axios from "axios";
import { useRouter } from "next/router";
import React, { FormEvent, useCallback, useEffect } from "react";
import PageLayout from "../components/PageLayout";
import { Button, Form, Input } from "@canonical/react-components";

export interface Response {
  data: {
    redirect_to?: string;
  };
}

function acceptUserCode(userCode: string, challenge: string) {
  axios
    .put("/api/device?device_challenge=" + String(challenge), {
      user_code: userCode,
    })
    .then(({ data }: Response) => {
      if (data.redirect_to) {
        window.location.href = data.redirect_to;
      }
    })
    .catch(() => {
      return Promise.reject();
    });
  return;
}

const DeviceCode: NextPage = () => {
  const router = useRouter();
  const { device_challenge: challenge, user_code: code } = router.query;

  useEffect(() => {
    // If the router is not ready yet, or we already have a flow, do nothing.
    if (!router.isReady) {
      return;
    }
  }, [code]);

  const handleSubmit = useCallback(
    (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault();

      const formData = new FormData(event.currentTarget);
      const userCode = formData.get("code");
      acceptUserCode(String(userCode), String(challenge));
    },
    [challenge],
  );

  return (
    <PageLayout title="Enter code to continue">
      <Form onSubmit={handleSubmit}>
        <Input
          id="code"
          name="code"
          type="text"
          placeholder="XXXXXXXX"
          autoFocus={true}
          defaultValue={code}
        />
        <Button
          type="submit"
          appearance="positive"
          className="u-no-margin--bottom"
        >
          Next
        </Button>
      </Form>
    </PageLayout>
  );
};

export default DeviceCode;
