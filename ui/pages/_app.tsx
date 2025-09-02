import "../static/sass/styles.scss";
import type { AppProps } from "next/app";
import dynamic from "next/dynamic";
import React from "react";

const ClientToastProvider = dynamic(
  async () => {
    const { ToastNotificationProvider } = await import(
      "@canonical/react-components"
    );
    return function Provider({ children }: { children: React.ReactNode }) {
      return <ToastNotificationProvider>{children}</ToastNotificationProvider>;
    };
  },
  { ssr: false },
);

export default function App({ Component, pageProps }: AppProps) {
  return (
    <ClientToastProvider>
      <Component {...pageProps} />
    </ClientToastProvider>
  );
}
