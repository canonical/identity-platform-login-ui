import { test, expect } from "@playwright/test";
import { enterTotpCode, setupTotp } from "./helpers/totp";
import { startGrafanaNewUserFlow } from "./helpers/grafana";
import { resetIdentities } from "./helpers/kratosIdentities";
import { USER_EMAIL, USER_PASSWORD, userPassLogin } from "./helpers/login";
import { confirmMailCode, enterNewPassword } from "./helpers/password";

const USER_PASSWORD_NEW = "abcABC123!!!";

test("reset password from grafana", async ({ context, page }) => {
  resetIdentities();
  await startGrafanaNewUserFlow(page);

  await expect(page.getByText("Sign in to grafana")).toBeVisible();
  await page.getByRole("link", { name: "Reset password" }).click();

  await page.getByLabel("Email").fill(USER_EMAIL);
  await expect(page).toHaveScreenshot({ fullPage: true, maxDiffPixels: 500 });
  await page.getByRole("button", { name: "Reset password" }).click();

  await confirmMailCode(page, context);
  await enterNewPassword(page, USER_PASSWORD_NEW);

  const totpSetupKey = await setupTotp(page);

  await expect(page.getByText("Account setup complete")).toBeVisible();
  await expect(page).toHaveScreenshot({ fullPage: true, maxDiffPixels: 500 });

  await context.clearCookies({ domain: "localhost" });
  await startGrafanaNewUserFlow(page);

  await userPassLogin(page, USER_EMAIL, USER_PASSWORD);
  await expect(page.getByText("Incorrect username or password")).toBeVisible();
  await expect(page).toHaveScreenshot({ fullPage: true, maxDiffPixels: 500 });

  await userPassLogin(page, USER_EMAIL, USER_PASSWORD_NEW);
  await enterTotpCode(page, totpSetupKey);

  await expect(page.getByText("Welcome to Grafana")).toBeVisible();
});
