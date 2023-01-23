import { Card, P, H3, CodeBox } from "@ory/themes"
import { AxiosError } from "axios"
import { useRouter } from "next/router"
import type { NextPage } from "next"
import { useEffect, useState } from "react"
import Head from "next/head"
import { kratos } from "../components/sdk"

const Home: NextPage = () => {
  const [session, setSession] = useState<string>(
    "No valid Ory Session was found.\nPlease sign in to receive one.",
  )
  const router = useRouter()

  useEffect(() => {
    kratos
      .toSession()
      .then(({ data }) => {
        setSession(JSON.stringify(data, null, 2))
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
  }, [router])

  return (
    <div className={"container-fluid"}>
      <Head>
        <title>Welcome to the Kratos UI</title>
      </Head>
      <Card wide>
        <div className="row">
          <div className="col-md-8 col-xs-12">
            <div className="box">
              <H3>Session Information</H3>
              <P>
                Below you will find the decoded Ory Session if you are logged
                in.
              </P>
              <CodeBox className="codebox" data-testid="session-content" code={session} />
            </div>
          </div>
        </div>
      </Card>
    </div>
  );
}

export default Home
