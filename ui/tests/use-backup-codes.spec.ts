import { test, expect } from "@playwright/test";
import { setupTotp } from "./helpers/totp";
import { startGrafanaNewUserFlow } from "./helpers/grafana";
import { resetIdentities } from "./helpers/kratosIdentities";
import { userPassLogin } from "./helpers/login";

test("backup recovery code setup and usage", async ({ context, page }) => {
  resetIdentities();
  await startGrafanaNewUserFlow(page);
  await userPassLogin(page);
  await setupTotp(context, page);

  await page.goto("http://localhost:4455/ui/setup_backup_codes");
  await page
    .getByRole("button", {
      name: "Generate new backup recovery codes",
      exact: true,
    })
    .click();

  const backupCode = await page.locator(".p-list__item").first().textContent();
  if (!backupCode) {
    throw new Error("Backup code not found");
  }

  await page
    .getByRole("button", { name: "Confirm backup recovery codes", exact: true })
    .click();

  await expect(page.getByText("Account setup complete")).toBeVisible();
  await expect(page).toHaveScreenshot({ fullPage: true, maxDiffPixels: 500 });

  await context.clearCookies({ domain: "localhost" });
  await startGrafanaNewUserFlow(page);
  await userPassLogin(page);

  await page
    .getByRole("button", { name: "Use backup code instead", exact: true })
    .click();

  await expect(page.getByText("Verify your identity")).toBeVisible();
  await expect(page).toHaveScreenshot({ fullPage: true, maxDiffPixels: 500 });
  await page.getByLabel("Backup recovery code").fill(backupCode);
  await page.getByRole("button", { name: "Sign in", exact: true }).click();

  await expect(page.getByText("Welcome to Grafana")).toBeVisible();
});
