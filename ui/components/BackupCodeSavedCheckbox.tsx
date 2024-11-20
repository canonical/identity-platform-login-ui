import { CheckboxInput } from "@canonical/react-components";
import React, { FC } from "react";

interface Props {
  isChecked: boolean;
  toggleChecked: () => void;
}

export const BackupCodeSavedCheckbox: FC<Props> = ({
  isChecked,
  toggleChecked,
}) => {
  console.log(isChecked);

  return (
    <div className="u-sv3 u-no-print checkbox-save-backup-codes">
      <CheckboxInput
        label="I saved the backup codes"
        onClick={toggleChecked}
        checked={isChecked}
      />
    </div>
  );
};
