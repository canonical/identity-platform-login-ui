import "../static/sass/styles.scss";
import type { AppProps } from "next/app";
import React from "react";
import ClientToastProvider from "../components/ClientToastProvider";
import { KratosSdkProvider } from "../api/kratosProvider";

export default function App({ Component, pageProps }: AppProps) {
  return (
    <KratosSdkProvider>
      <ClientToastProvider>
        <Component {...pageProps} />
      </ClientToastProvider>
    </KratosSdkProvider>
  );
}
