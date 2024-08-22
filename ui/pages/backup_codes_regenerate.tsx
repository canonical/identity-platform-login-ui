import type { NextPage } from "next";
import React from "react";
import PageLayout from "../components/PageLayout";
import { Button } from "@canonical/react-components";

const BackupCodesRegenerate: NextPage = () => {
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
        <Button onClick={() => (window.location.href = "./setup_backup_codes")}>
          Generate new codes
        </Button>
        <Button
          onClick={() => (window.location.href = "./login")}
          appearance="link"
        >
          I don&apos;t need new codes, sign in
        </Button>
      </div>
    </PageLayout>
  );
};

export default BackupCodesRegenerate;
