import { NextRouter } from "next/router";

export function redirectTo(url: string, router: NextRouter): void {
  const newUrl = new URL(url);
  const kratosParams = Object.fromEntries(newUrl.searchParams.entries());
  const basePath = router.basePath || "";
  const pathWithoutBase = newUrl.pathname.startsWith(basePath)
    ? newUrl.pathname.slice(basePath.length)
    : newUrl.pathname;
  void router.push({
    pathname: pathWithoutBase,
    query: {
      ...router.query,
      ...kratosParams,
    },
  });
}