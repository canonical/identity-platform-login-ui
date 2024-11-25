import { Button, List, Modal } from "@canonical/react-components";
import React, { FC } from "react";

interface Props {
  hasModal: boolean;
  onCancel: () => void;
  onConfirm: () => void;
}

export const BackupCodeDeletionModal: FC<Props> = ({
  hasModal,
  onCancel,
  onConfirm,
}) => {
  return (
    <>
      {hasModal && (
        <Modal
          buttonRow={
            <>
              <Button
                className="u-no-margin--bottom"
                onClick={(e) => {
                  e.preventDefault();
                  onCancel();
                }}
                appearance="base"
              >
                Cancel
              </Button>
              <Button
                appearance="negative"
                className="u-no-margin--bottom"
                onClick={(e) => {
                  e.preventDefault();
                  onConfirm();
                }}
              >
                Deactivate backup codes
              </Button>
            </>
          }
          title="Deactivate backup codes"
        >
          You{"'"}re about to deactivate your backup codes. Please be aware of
          the following:
          <List
            className="deactivate-backup-codes-list"
            items={[
              "All your current backup codes will no longer work.",
              "You won't be able to use backup codes to access your account if you lose access to your primary authentication method.",
              "You'll need to set up new backup codes if you want to use them again in the future.",
            ]}
          />
        </Modal>
      )}
    </>
  );
};
