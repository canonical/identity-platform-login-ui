import { expect, Page } from "@playwright/test";
import { getRecoveryCodeFromMailSlurp } from "./mail";
import { BrowserContext } from "playwright-core";

export const confirmMailCode = async (
  page: Page,
  context: BrowserContext,
) => {
  await expect(page.getByText("Enter the code you received")).toBeVisible();
  await expect(page).toHaveScreenshot({ fullPage: true, maxDiffPixels: 500 });
  const recoveryCode = await getRecoveryCodeFromMailSlurp(context);
  await page.getByLabel("Recovery code").fill(recoveryCode);
  await page.getByRole("button", { name: "Submit" }).click();
};

export const enterNewPassword = async (page: Page, password: string) => {
  await expect(page.getByText("Reset password").first()).toBeVisible();
  await expect(page).toHaveScreenshot({ fullPage: true, maxDiffPixels: 500 });
  await page.getByLabel("New password", { exact: true }).fill(password);
  await page.getByLabel("Confirm New password").fill(password);
  await page.getByRole("button", { name: "Reset password" }).click();
};
