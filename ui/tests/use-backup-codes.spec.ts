import { test, expect } from "@playwright/test";
import { setupTotp } from "./helpers/totp";
import { startGrafanaNewUserFlow } from "./helpers/grafana";
import { resetIdentities } from "./helpers/kratosIdentities";
import { userPassLogin } from "./helpers/login";
import { clickButton, verifyBackupCode } from "./helpers/backupCode";

test("backup recovery code setup and usage", async ({ context, page }) => {
  resetIdentities();
  await startGrafanaNewUserFlow(page);
  await userPassLogin(page);
  await setupTotp(page);

  await page.goto("http://localhost:4455/ui/setup_backup_codes");
  await clickButton(page, "Generate new backup recovery codes");

  const backupCode = await page.locator(".p-list__item").first().textContent();
  if (!backupCode) {
    throw new Error("Backup code not found");
  }

  await clickButton(page, "Confirm backup recovery codes");

  await expect(page.getByText("Account setup complete")).toBeVisible();
  await expect(page).toHaveScreenshot({ fullPage: true, maxDiffPixels: 500 });

  await context.clearCookies({ domain: "localhost" });
  await startGrafanaNewUserFlow(page);
  await userPassLogin(page);

  await clickButton(page, "Use backup code instead");
  await verifyBackupCode(page, backupCode);

  await expect(page.getByText("Welcome to Grafana")).toBeVisible();
});
