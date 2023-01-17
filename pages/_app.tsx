
import "../styles/globals.css"
import type { AppProps } from "next/app"
import { ToastContainer } from "react-toastify"
import Logo from "../components/Logo"


function MyApp({ Component, pageProps }: AppProps) {
  return (
    <div data-testid="app-react">
      <Logo />
      <Component {...pageProps} />
      <ToastContainer />
    </div>
  )
}

export default MyApp