import type { NextPage } from "next";
import React, { useEffect, useState } from "react";
import PageLayout from "../components/PageLayout";
import { Button } from "@canonical/react-components";
import { useRouter } from "next/router";
import { kratos } from "../api/kratos";
import { handleFlowError } from "../util/handleFlowError";
import { LoginFlow } from "@ory/client";

const BackupCodesRegenerate: NextPage = () => {
  const [flow, setFlow] = useState<LoginFlow>();

  const router = useRouter();
  const { flow: flowId } = router.query;

  useEffect(() => {
    // If the router is not ready yet, or we already have a flow, do nothing.
    if (!router.isReady || flow) {
      return;
    }

    if (flowId) {
      kratos
        .getLoginFlow({ id: String(flowId) })
        .then((res) => setFlow(res.data))
        .catch(handleFlowError("login", setFlow));
      return;
    }
  }, [flowId, router, router.isReady, flow]);

  const signInUrl = flow?.return_to ?? "./login";

  return (
    <PageLayout title="Backup code sign in successful">
      <p className="u-text--muted">
        You&apos;ve just used a backup code. Would you like to generate new ones
        to ensure you have a full set?
      </p>
      <p className="u-text--muted">
        Generating new codes will invalidate all previous codes.
      </p>
      <div>
        <Button
          onClick={() =>
            (window.location.href = `./setup_backup_codes?return_to=${flow?.return_to}`)
          }
        >
          Generate new codes
        </Button>
        <Button
          onClick={() => (window.location.href = signInUrl)}
          appearance="link"
        >
          I don&apos;t need new codes, sign in
        </Button>
      </div>
    </PageLayout>
  );
};

export default BackupCodesRegenerate;
