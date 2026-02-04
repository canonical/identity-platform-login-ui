import { test, expect } from "@playwright/test";
import { setupTotp } from "./helpers/totp";
import { finishAuthFlow, startNewAuthFlow } from "./helpers/oidc_client";
import { resetIdentities } from "./helpers/kratosIdentities";
import { userPassLogin } from "./helpers/login";
import { clickButton, verifyBackupCode } from "./helpers/backupCode";

test("backup recovery code setup and usage", async ({ browser, context, page }) => {
  resetIdentities();
  await startNewAuthFlow(page);
  await userPassLogin(page);
  await setupTotp(page);
  await finishAuthFlow(page);

  await page.goto("http://localhost/ui/setup_backup_codes");
  await clickButton(page, "Create backup codes");

  const backupCode = await page.locator(".p-list__item").first().textContent();
  if (!backupCode) {
    throw new Error("Backup code not found");
  }

  await page.getByText("I saved the backup codes").click();
  await clickButton(page, "Create backup codes");

  await expect(page.getByText("Account setup complete")).toBeVisible();

  // Start login in a new context as user is already authenticated within the current context
  const newContext = await browser.newContext();
  const newPage = await newContext.newPage();

  await startNewAuthFlow(newPage);
  await userPassLogin(newPage);

  await clickButton(newPage, "Use backup code instead");
  await verifyBackupCode(newPage, backupCode);

  await finishAuthFlow(newPage);
});
