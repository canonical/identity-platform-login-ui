import Image from "next/image"

export default function Logo(){
    return (
      <div className="center-sm">
        <Image src={ "/logo-canonical-aubergine.svg" } alt="" />
      </div>
    )
}