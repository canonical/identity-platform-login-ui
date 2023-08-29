import Image from "next/image";
import React, { FC } from "react";

const Logo: FC = () => (
  <div className="p-panel__logo canonical-logo">
    <Image src={"./logo-canonical.svg"} alt="Canonical" />
  </div>
);

export default Logo;
