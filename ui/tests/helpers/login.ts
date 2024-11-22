import { expect, Page } from "@playwright/test";

export const USER_EMAIL = "test@example.com";
export const USER_PASSWORD = "test";

export const userPassLogin = async (
  page: Page,
  email: string = USER_EMAIL,
  password: string = USER_PASSWORD,
) => {
  await expect(page).toHaveScreenshot({ fullPage: true, maxDiffPixels: 500 });
  await page.getByLabel("E-Mail").fill(email);
  await page.getByLabel("Password").fill(password);
  await page.getByRole("button", { name: "Sign in", exact: true }).click();
};
