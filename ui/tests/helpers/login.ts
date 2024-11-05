import { expect, Page } from "@playwright/test";

export const USER_EMAIL = "test@example.com";
export const USER_PASS = "test";

export const userPassLogin = async (page: Page) => {
  await expect(page.getByText("Sign in to grafana")).toBeVisible();
  await expect(page).toHaveScreenshot({ fullPage: true });
  await page.getByLabel("E-Mail").fill(USER_EMAIL);
  await page.getByLabel("Password").fill(USER_PASS);
  await page.getByRole("button", { name: "Sign in", exact: true }).click();
};
