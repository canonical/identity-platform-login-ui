import "../static/css/styles.css";
import type { AppProps } from "next/app";
import { useRouter } from "next/router";
import { ToastContainer } from "react-toastify";
import Logo from "../components/Logo";
import React, { FC } from "react";

const App: FC<AppProps> = ({ Component, pageProps }) => {
  const router = useRouter();
  return (
    <div data-testid="app-react">
      {router.pathname !== "/consent" ? <Logo /> : null}
      <Component {...pageProps} />
      <ToastContainer />
    </div>
  );
};

export default App;
