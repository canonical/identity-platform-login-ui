import type { NextPage } from "next"
import { AxiosError } from "axios"
import { useEffect } from "react"
import { useRouter } from "next/router"
import { hydraAdmin } from "../components/hydra"
import { kratos } from "../components/sdk"
import {Identity} from "@ory/client"


function exportClaims({traits}: Identity, scopes) {
  const scopes_to_claims = {
    openid: ["sub"],
    profile:  [
      "name",
      "family_name",
      "given_name",
      "middle_name",
      "nickname",
      "preferred_username",
      "profile",
      "picture",
      "website",
      "gender",
      "birthdate",
      "zoneinfo",
      "locale",
      "updated_at"
    ],
    email: ["email", "email_verified"],
    address: ["address"],
    phone: ["phone_number", "phone_number_verified"]
  }

  const claims = scopes.reduce(
    (res, key) => {
      if (!(key in scopes_to_claims)) {
        return res
      }
      return [...res, ...scopes_to_claims[key]]
    }, []
  )

  return claims.reduce((res, key) => {
    res[key] = traits[key]
    return res
  }, {}
  )
}


const Consent: NextPage = () => {
  const router = useRouter()
  const { consent_challenge } = router.query;

  useEffect(() => {
    if (!router.isReady) {
      return
    }

    kratos
    .toSession()
    .then(({ data: user_identity }) => {
      hydraAdmin
      .getOAuth2ConsentRequest({
        consentChallenge: consent_challenge ? String(consent_challenge) : undefined,
      })
      // This will be called if the HTTP request was successful
      .then(({ data: body }) => {
        // If a user has granted this application the requested scope, hydra will tell us to not show the UI.
        const skip = true || body.skip
        if (skip) {

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
                  id_token: exportClaims(user_identity.identity, body.requested_scope),
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
    })
    .catch((err: AxiosError) => {
      switch (err.response?.status) {
        case 403:
          // This is a legacy error code thrown. See code 422 for
          // more details.
          return router.push("/login?aal=aal2")
        case 422:
          // This status code is returned when we are trying to
          // validate a session which has not yet completed
          // its second factor
          return router.push("/login?aal=aal2")
        case 401:
          // do nothing, the user is not logged in
          return
      }

      // Something else happened!
      return Promise.reject(err)
    })
    }, [router, consent_challenge])

    return (<></>)
}

export default Consent
