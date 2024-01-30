import "../static/sass/styles.scss";
import type { AppProps } from "next/app";
import { ToastContainer } from "react-toastify";
import React, { FC } from "react";

const App: FC<AppProps> = ({ Component, pageProps }) => {
  return (
    <>
      <Component {...pageProps} />
      <ToastContainer />
    </>
  );
};

export default App;
