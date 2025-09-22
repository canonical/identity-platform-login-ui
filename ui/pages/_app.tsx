import "../static/sass/styles.scss";
import type { AppProps } from "next/app";
import React from "react";
import ClientToastProvider from "../components/ClientToastProvider";

export default function App({ Component, pageProps }: AppProps) {
  return (
    <ClientToastProvider>
      <Component {...pageProps} />
    </ClientToastProvider>
  );
}
