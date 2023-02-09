import Image from "next/image"

export default function Logo(){
    return (
      <div className="p-panel__logo u-align--center">
        <Image src={ "/logo-canonical-aubergine.svg" } alt="" />
      </div>
    )
}