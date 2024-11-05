import { test, expect } from "@playwright/test";
import { getTotpCode, setupTotp } from "./helpers/totp";
import { startGrafanaNewUserFlow } from "./helpers/grafana";
import { resetIdentities } from "./helpers/kratosIdentities";
import { getRecoveryCodeFromMailSlurp } from "./helpers/mail";
import { USER_EMAIL, userPassLogin } from "./helpers/login";

const PASSWORD_NEW = "abcABC123!!!";

test("reset password from grafana", async ({ context, page }) => {
  resetIdentities();
  await startGrafanaNewUserFlow(page);

  await expect(page.getByText("Sign in to grafana")).toBeVisible();
  await page.getByRole("link", { name: "Reset password" }).click();

  await page.getByLabel("Email").fill(USER_EMAIL);
  await expect(page).toHaveScreenshot({ fullPage: true });
  await page.getByRole("button", { name: "Reset password" }).click();

  await expect(page.getByText("Enter the code you received")).toBeVisible();
  await expect(page).toHaveScreenshot({ fullPage: true });
  const recoveryCode = await getRecoveryCodeFromMailSlurp(context, page);
  await page.getByLabel("Recovery code").fill(recoveryCode);
  await page.getByRole("button", { name: "Submit" }).click();

  await expect(page.getByText("Reset password").first()).toBeVisible();
  await expect(page).toHaveScreenshot({ fullPage: true });
  await page.getByLabel("New password", { exact: true }).fill(PASSWORD_NEW);
  await page.getByLabel("Confirm New password").fill(PASSWORD_NEW);
  await page.getByRole("button", { name: "Reset password" }).click();

  const totpPage = await setupTotp(context, page);

  await expect(page.getByText("Account setup complete")).toBeVisible();
  await expect(page).toHaveScreenshot({ fullPage: true });

  await context.clearCookies({ domain: "localhost" });
  await startGrafanaNewUserFlow(page);

  await userPassLogin(page);
  await expect(page.getByText("Incorrect username or password")).toBeVisible();
  await expect(page).toHaveScreenshot({ fullPage: true });

  await page.getByLabel("Password").fill(PASSWORD_NEW);
  await page.getByRole("button", { name: "Sign in", exact: true }).click();

  await expect(page.getByText("Verify your identity")).toBeVisible();
  await expect(page).toHaveScreenshot({ fullPage: true });
  const totpCode = await getTotpCode(totpPage);
  await page.getByLabel("Authentication code").fill(totpCode);
  await page.getByRole("button", { name: "Sign in" }).click();

  await expect(page.getByText("Welcome to Grafana")).toBeVisible();
});
