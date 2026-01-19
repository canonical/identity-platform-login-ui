import { expect, Page } from "@playwright/test";

export const clickButton = async (page: Page, name: string) => {
  await page
    .getByRole("button", {
      name,
      exact: true,
    })
    .click();
};

export const verifyBackupCode = async (page: Page, backupCode: string) => {
  await expect(page.getByText("Verify your identity")).toBeVisible();
  await page.getByLabel("Backup recovery code").fill(backupCode);
  await page.getByRole("button", { name: "Sign in", exact: true }).click();
};
