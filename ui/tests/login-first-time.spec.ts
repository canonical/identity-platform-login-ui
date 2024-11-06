import { test, expect } from "@playwright/test";
import { setupTotp } from "./helpers/totp";
import { startGrafanaNewUserFlow } from "./helpers/grafana";
import { resetIdentities } from "./helpers/kratosIdentities";
import { userPassLogin } from "./helpers/login";

test("first time login to grafana", async ({ context, page }) => {
  resetIdentities();
  await startGrafanaNewUserFlow(page);
  await userPassLogin(page);
  await setupTotp(context, page);

  await expect(page.getByText("Account setup complete")).toBeVisible();
  await expect(page).toHaveScreenshot({ fullPage: true, maxDiffPixels: 500 });

  await expect(page.getByText("Welcome to Grafana")).toBeVisible();
});
