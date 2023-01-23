import type { NextPage } from "next"
import { useEffect, useState } from "react"
import { useRouter } from "next/router"
import { hydraAdmin } from "../components/hydra"


const Consent: NextPage = () => {
  const router = useRouter()
  const { consent_challenge } = router.query;

  useEffect(() => {
    if (!router.isReady) {
      return
    }

    hydraAdmin
      .getOAuth2ConsentRequest({
        consentChallenge: consent_challenge ? String(consent_challenge) : undefined,
      })
      // This will be called if the HTTP request was successful
      .then(({ data: body }) => {
        // If a user has granted this application the requested scope, hydra will tell us to not show the UI.
        const skip = true || body.skip
        if (skip) {
          // You can apply logic here, for example grant another scope, or do whatever...
          // ...

          // Now it's time to grant the consent request. You could also deny the request if something went terribly wrong
          return hydraAdmin
            .acceptOAuth2ConsentRequest({
              consentChallenge: consent_challenge ? String(consent_challenge) : undefined,
              acceptOAuth2ConsentRequest: {
                // We can grant all scopes that have been requested - hydra already checked for us that no additional scopes
                // are requested accidentally.
                grant_scope: body.requested_scope,
                // ORY Hydra checks if requested audiences are allowed by the client, so we can simply echo this.
                grant_access_token_audience: body.requested_access_token_audience,
                // The session allows us to set session data for id and access tokens
                session: {
                  access_token: null,
                  id_token: null,
                },
              }
            })
            .then(({ data: body2 }) => {
              // All we need to do now is to redirect the user back to hydra!
              window.location.href = body2.redirect_to
            })
        }
      })
      // This will handle any error that happens when making HTTP calls to hydra
      .catch((err) => {
        console.log(err)
      })
    }, [router])

    return (<></>)
}

export default Consent
