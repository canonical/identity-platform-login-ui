import { Card, Row } from "@canonical/react-components";
import axios from "axios";
import Head from "next/head";
import { useRouter } from "next/router";
import React, { FormEvent, useCallback, useEffect } from "react";

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

export default function Page() {
  const router = useRouter();
  const { device_challenge: challenge, user_code: code } = router.query;

  useEffect(() => {
    // If the router is not ready yet, or we already have a flow, do nothing.
    if (!router.isReady) {
      return;
    }
    if (code != null) {
      acceptUserCode(String(code), String(challenge));
    }
  }, [code]);

  const handleSubmit = useCallback(
    (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault();

      const formData = new FormData(event.currentTarget);
      const userCode = formData.get("user_code");
      acceptUserCode(String(userCode), String(challenge));
    },
    [challenge],
  );

  return (
    <>
      <Head>
        <title>Enter code</title>
      </Head>
      <Row className="p-strip is-shallow">
        <div className="login-card">
          <div>
            <Card title="Enter your code">
              <form onSubmit={handleSubmit}>
                <input type="text" name="user_code" />
                <button type="submit">Submit</button>
              </form>
            </Card>
          </div>
        </div>
      </Row>
    </>
  );
}
