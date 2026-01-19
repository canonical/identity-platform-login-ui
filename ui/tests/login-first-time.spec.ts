import { test, expect } from "@playwright/test";
import { setupTotp } from "./helpers/totp";
import { resetIdentities } from "./helpers/kratosIdentities";
import { userPassLogin } from "./helpers/login";
import { finishAuthFlow, startNewAuthFlow } from "./helpers/oidc_client";

test("first time login to oidc app", async ({ page }) => {
  resetIdentities();

  await startNewAuthFlow(page);
  await userPassLogin(page);
  await setupTotp(page);

  await expect(page.getByText("Account setup complete")).toBeVisible();

  await finishAuthFlow(page);
});
