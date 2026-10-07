import { test, expect, Page } from "@playwright/test";
import { enterTotpCode, setupTotp } from "./helpers/totp";
import { finishAuthFlow, startNewAuthFlow } from "./helpers/oidc_client";
import { resetIdentities } from "./helpers/kratosIdentities";
import { userPassLogin } from "./helpers/login";

// The flow of the previous step can no longer be used, so the login starts
// again. It must still be the login the client is waiting for.
const backToNewLogin = async (page: Page) => {
  await page.goBack();
  await expect(page.getByLabel("Email")).toBeVisible();
  await expect(page).toHaveURL(/login_challenge=/);
};

test("browser back on the authenticator setup keeps the login of the oidc client", async ({
  page,
}) => {
  resetIdentities();
  await startNewAuthFlow(page);
  await userPassLogin(page);
  await expect(page.getByText("Secure your account")).toBeVisible();

  await backToNewLogin(page);

  await userPassLogin(page);
  await setupTotp(page);
  await expect(page.getByText("Account setup complete")).toBeVisible();
  await finishAuthFlow(page);
});

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

  await backToNewLogin(newPage);

  await userPassLogin(newPage);
  await enterTotpCode(newPage, setupKey);
  await finishAuthFlow(newPage);
});
