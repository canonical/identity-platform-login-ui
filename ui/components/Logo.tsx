import Image from "next/image";
import React, { FC } from "react";
import { Row } from "@canonical/react-components";

const Logo: FC = () => (
  <Row>
    <div className="p-panel__logo u-align--center">
      <Image src={"./logo-canonical-aubergine.svg"} alt="Canonical logo" />
    </div>
  </Row>
);

export default Logo;
