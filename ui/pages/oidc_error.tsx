import React, { useState, useEffect } from 'react';
import { Card } from "@canonical/react-components";
import type { NextPage } from "next"
import { useRouter } from "next/router"


const OIDCError: NextPage = () => {
  const [error_msg, setErrorMsg] = useState<string>("");
  const router = useRouter()
  const { error, error_description } = router.query

  useEffect(() => {
    // If the router is not ready yet, or we already have an error, do nothing.
    if (!router.isReady) {
      return
    }

    setErrorMsg(JSON.stringify({
        error: error,
        error_description: error_description
      },
      null,
      2
    ))
  }, [error, error_description, router, router.isReady])

  return (
    <>
      <Card title="An error occurred" >
        <div>
          <pre className="codebox" >{error_msg}</pre>
        </div>
      </Card>
    </>
  )
}

export default OIDCError
