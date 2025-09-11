import dynamic from "next/dynamic";
import React, { PropsWithChildren } from "react";

const ClientToastProvider = dynamic(
  async () => {
    const { ToastNotificationProvider } = await import(
      "@canonical/react-components"
    );
    return function Provider({ children }: PropsWithChildren) {
      return <ToastNotificationProvider>{children}</ToastNotificationProvider>;
    };
  },
  { ssr: false },
);

export default ClientToastProvider;
