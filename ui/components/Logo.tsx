import Image from "next/image";
import React, { FC } from "react";

const Logo: FC = () => (
  <div className="p-panel__logo u-align--center">
    <Image src={"./logo-canonical-aubergine.svg"} alt="" />
  </div>
);

export default Logo;
