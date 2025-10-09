import { Configuration, FrontendApi } from "@ory/client";
import React, { createContext, useContext, useEffect, useState } from "react";

const defaultkratos = new FrontendApi(
  new Configuration({
    basePath: "..",
    baseOptions: {
      withCredentials: true,
    },
  })
);

interface KratosSdkContextType {
  kratos: FrontendApi;
  kratosReady: boolean;
}

const KratosSdkContext = createContext<KratosSdkContextType>({kratos: defaultkratos, kratosReady: false});

export function useKratos() {
  return useContext(KratosSdkContext);
}

export function KratosSdkProvider({ children }: { children: React.ReactNode }) {
  const [kratos, setkratos] = useState(defaultkratos);
  const [kratosReady, setKratosReady] = useState(false);

  useEffect(() => {
    (async () => {
      try {
        const res = await fetch("../api/v0/app-config");
        const data = await res.json();
        const basePath = data.kratos_base_path ?? "..";

        setkratos(
          new FrontendApi(
            new Configuration({
              basePath,
              baseOptions: { withCredentials: true },
            })
          )
        );

        setKratosReady(true); // Set ready *after* kratos is set
      } catch (err) {
        console.error("Failed to initialize Kratos SDK", err);
      }
    })();
  }, []);

  return (
    <KratosSdkContext.Provider value={{ kratos, kratosReady }}>
      {children}
    </KratosSdkContext.Provider>
  );
}
