import { Button } from "@canonical/react-components";
import React, { FC } from "react";
import { useRouter } from "next/router";

export const UseBackupCodeButton: FC = () => {
  const router = useRouter();

  return (
    <Button
      appearance="link"
      tabIndex={5}
      type="button"
      onClick={() => {
        const newQuery = { ...router.query, ["use_backup_code"]: "true" };
        void router.push(
          {
            pathname: window.location.pathname,
            query: newQuery,
          },
          undefined,
          { shallow: true },
        );
      }}
    >
      Use backup code instead
    </Button>
  );
};
