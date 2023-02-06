import type { NextPage } from "next"
import { AxiosError } from "axios"
import get from "axios"
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

    get("/api/consent?consent_challenge=" + consent_challenge)
    .then(({ data : body}) => {
      window.location.href = body.redirect_to
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
