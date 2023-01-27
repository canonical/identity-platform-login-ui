import { CardTitle, Card } from "@ory/themes"
import type { NextPage } from "next"
import { useRouter } from "next/router"
import { useEffect } from "react"


const OIDCError: NextPage = () => {
  const router = useRouter()
  const { error, error_description } = router.query
  const error_msg = {
    error: error,
    error_description: error_description
  }

  useEffect(() => {
    // If the router is not ready yet, or we already have an error, do nothing.
    if (!router.isReady) {
      return
    }

    console.log(JSON.stringify(error_msg, null, 2))
  }, [error, error_description, error_msg, router, router.isReady])

  return (
    <>
      <Card wide >
        <CardTitle>An error occurred</CardTitle>
        <div>
          <pre className="codebox" >{JSON.stringify(error_msg, null, 2)}</pre>
        </div>
      </Card>
    </>
  )
}

export default OIDCError
