import type { NextPage } from "next";
import { AxiosError } from "axios";
import get from "axios";
import { useEffect } from "react";
import { useRouter } from "next/router";
import React from "react";
import { LoginFlow } from "@ory/client";
import { KratosErrorResponse } from "../util/handleFlowError";

export interface FlowResponse {
  data: {
    redirect_to?: string;
  } & LoginFlow;
}

const Consent: NextPage = () => {
  const router = useRouter();
  const { consent_challenge } = router.query;

  useEffect(() => {
    if (!router.isReady) {
      return;
    }

    get(`../api/consent?consent_challenge=${consent_challenge as string}`)
      .then(({ data }: FlowResponse) => {
        if (data.redirect_to) {
          window.location.href = data.redirect_to;
        }
      })
      .catch((err: AxiosError) => {
        const response = err.response?.data as KratosErrorResponse;
        switch (err.response?.status) {
          case 403:
            // This is a legacy error code thrown. See code 422 for
            // more details.
            window.location.href = "./login?aal=aal2";
            return;
          case 422:
            // This status code is returned when we are trying to
            // validate a session which has not yet completed
            // its second factor
            window.location.href = "./login?aal=aal2";
            return;
          case 401:
            // do nothing, the user is not logged in
            return;
          case 303:
            // This status is returned when user must be redirected
            // to set up 2fa
            if (response.error?.id == "session_aal2_required") {
              window.location.href = response.redirect_browser_to;
              return;
            }
            return;
        }

        // Something else happened!
        return Promise.reject(err);
      });
  }, [router, consent_challenge]);

  return <></>;
};

export default Consent;
