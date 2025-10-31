import { test, expect } from "@playwright/test";
import { enterTotpCode, setupTotp } from "./helpers/totp";
import { finishAuthFlow, startNewAuthFlow } from "./helpers/oidc_client";
import { resetIdentities } from "./helpers/kratosIdentities";
import { USER_EMAIL, USER_PASSWORD, userPassLogin } from "./helpers/login";
import { confirmMailCode, enterNewPassword } from "./helpers/password";
import { SCREENSHOT_OPTIONS } from "./helpers/visual";

const USER_PASSWORD_NEW = "abcABC123!!!";

test("reset password from oidc app", async ({ context, page }) => {
  resetIdentities();
  await startNewAuthFlow(page);
  const emailInput = page.getByLabel("Email");

  await emailInput.fill(USER_EMAIL);
  await page.getByRole("button", { name: "Continue", exact: true }).click();
  await page.getByRole("link", { name: "Reset password" }).click();

  await page.getByLabel("Email").fill(USER_EMAIL);
  await expect(page).toHaveScreenshot(SCREENSHOT_OPTIONS);
  await page.getByRole("button", { name: "Reset password" }).click();

  await confirmMailCode(page, context);
  await enterNewPassword(page, USER_PASSWORD_NEW);

  const totpSetupKey = await setupTotp(page);

  await expect(page.getByText("Account setup complete")).toBeVisible();
  await expect(page).toHaveScreenshot(SCREENSHOT_OPTIONS);
  await finishAuthFlow(page);

  await startNewAuthFlow(page);

  await userPassLogin(page, USER_EMAIL, USER_PASSWORD);
  await expect(page.getByText("Incorrect username or password")).toBeVisible();
  await expect(page).toHaveScreenshot(SCREENSHOT_OPTIONS);

  await userPassLogin(page, USER_EMAIL, USER_PASSWORD_NEW);
  await enterTotpCode(page, totpSetupKey);

  await finishAuthFlow(page);
});
