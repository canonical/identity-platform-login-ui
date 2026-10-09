import { NextRouter } from "next/router";
import { useCallback, useEffect, useState } from "react";

export function redirectTo(
  url: string,
  router: NextRouter,
  passQuery: boolean = true,
): void {
  let newUrl: URL;
  try {
    newUrl = new URL(url, window.location.origin);
  } catch (error) {
    console.error(`Failed to redirect to invalid URL: ${url}`, error);
    return;
  }
  if (newUrl.origin !== window.location.origin) {
    window.location.assign(newUrl.toString());
    return;
  }
  const kratosParams = Object.fromEntries(newUrl.searchParams.entries());
  const basePath = router.basePath || "";
  const pathWithoutBase = newUrl.pathname.startsWith(basePath)
    ? newUrl.pathname.slice(basePath.length)
    : newUrl.pathname;

  void router.push({
    pathname: pathWithoutBase,
    query: {
      ...(passQuery ? router.query : {}),
      ...kratosParams,
    },
    hash: newUrl.hash,
  });
}

// A redirect the backend answers with. redirect_label is set when the
// destination is a company sign-in.
type LabelledRedirect = {
  redirect_to?: string;
  redirect_label?: string;
};

export const getRedirectLabel = (data: unknown): string | undefined => {
  const label = (data as LabelledRedirect | undefined)?.redirect_label;
  return typeof label === "string" && label !== "" ? label : undefined;
};

// useLabelledRedirect returns the label of the company sign-in being
// redirected to (render "Redirecting to <label>…" while it is set) and the
// function that follows a redirect. Without a label the browser leaves at
// once; with one, it leaves only after the step has been rendered.
export const useLabelledRedirect = (): [
  string | undefined,
  (url: string, label?: string) => void,
] => {
  const [pending, setPending] = useState<{ url: string; label: string }>();

  useEffect(() => {
    if (pending) {
      window.location.href = pending.url;
    }
  }, [pending]);

  const redirect = useCallback((url: string, label?: string) => {
    if (!label) {
      window.location.href = url;
      return;
    }
    setPending({ url, label });
  }, []);

  return [pending?.label, redirect];
};
