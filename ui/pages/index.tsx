import { Row, CodeSnippet } from "@canonical/react-components";
import { AxiosError } from "axios"
import { useRouter } from "next/router"
import type { NextPage } from "next"
import { useEffect, useState } from "react"
import { kratos } from "../components/sdk"

const Home: NextPage = () => {
  const [session, setSession] = useState<string>(
    "No valid Session was found.\nPlease sign in to receive one.",
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
    <div className="p-code-snippet">
      <Row >
              <CodeSnippet blocks={[{
                title: "Session Information",
                wrapLines: true,
                code: session
              }]} />
              </Row>
    </div>
  );
}

export default Home
