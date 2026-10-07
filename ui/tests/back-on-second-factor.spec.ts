import { test, expect } from "@playwright/test";
import { enterTotpCode, setupTotp } from "./helpers/totp";
import { finishAuthFlow, startNewAuthFlow } from "./helpers/oidc_client";
import { resetIdentities } from "./helpers/kratosIdentities";
import { userPassLogin } from "./helpers/login";

test("browser back on the second factor keeps the login of the oidc client", async ({
  browser,
  page,
}) => {
  resetIdentities();
  await startNewAuthFlow(page);
  await userPassLogin(page);
  const setupKey = await setupTotp(page);
  await finishAuthFlow(page);

  // Start login in a new context as user is already authenticated within the current context
  const newContext = await browser.newContext();
  const newPage = await newContext.newPage();

  await startNewAuthFlow(newPage);
  await userPassLogin(newPage);
  await expect(
    newPage.getByRole("heading", { name: "Verify your identity" }),
  ).toBeVisible();

  // The flow of the previous step can no longer be used, so the login starts
  // again. It must still be the login the client is waiting for.
  await newPage.goBack();
  await expect(newPage.getByLabel("Email")).toBeVisible();

  await userPassLogin(newPage);
  await enterTotpCode(newPage, setupKey);
  await finishAuthFlow(newPage);
});
