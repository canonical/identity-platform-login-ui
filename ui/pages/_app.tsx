import "../static/sass/styles.scss";
import type { AppProps } from "next/app";
import React from "react";
import ClientToastProvider from "../components/ClientToastProvider";
import { AppConfigProvider } from "../config/useAppConfig";

export default function App({ Component, pageProps }: AppProps) {
  return (
    <AppConfigProvider>
      <ClientToastProvider>
        <Component {...pageProps} />
      </ClientToastProvider>
    </AppConfigProvider>
  );
}
