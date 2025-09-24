import { expect, Page } from "@playwright/test";

export const USER_EMAIL = "test@example.com";
export const USER_PASSWORD = "test";

export const userPassLogin = async (
  page: Page,
  email: string = USER_EMAIL,
  password: string = USER_PASSWORD,
) => {
  await page.getByLabel("Email").fill(email);
  await page.getByRole("button", { name: "Continue", exact: true }).click();

  const passwordInput = page.getByRole("textbox", {name: "Password"});
  await expect(passwordInput).toBeVisible();
  await passwordInput.fill(password);
  await page.getByRole("button", { name: "Sign in", exact: true }).click();
};
