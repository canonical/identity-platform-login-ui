import { randomNameSuffix } from "./name";
import { BrowserContext } from "playwright-core";
import { expect, Page } from "@playwright/test";

export const getTotpCode = async (totpPage: Page) => {
  const totpCode = await totpPage.locator(".code").textContent();
  if (!totpCode) {
    throw new Error("TOTP code not found");
  }
  return totpCode;
};

export const setupTotp = async (context: BrowserContext, page: Page) => {
  await expect(page.getByText("Secure your account")).toBeVisible();
  await expect(page).toHaveScreenshot({ fullPage: true, maxDiffPixelRatio: 0.05 }); // the code differs every time
  const setupCode = await getTotpSetupCode(page);
  const totpPage = await setupTotpApp(context, setupCode);
  const totpCode = await getTotpCode(totpPage);
  await page.getByLabel("Verify code").fill(totpCode);
  await page.getByRole("button", { name: "Save" }).click();
  return totpPage;
};

const getTotpSetupCode = async (page: Page) => {
  const secretKey = await page.locator("pre").textContent();
  if (!secretKey) {
    throw new Error("TOTP setup code not found");
  }
  return secretKey;
};

const setupTotpApp = async (context: BrowserContext, setupCode: string) => {
  const appName = `app-${randomNameSuffix()}`;

  const totpPage = await context.newPage();
  await totpPage.goto("https://totp.app/");
  await totpPage.getByTitle("Add").click();
  await totpPage.getByPlaceholder("Secret key").fill(setupCode);
  await totpPage.getByPlaceholder("Application name").fill(appName);
  await totpPage.getByRole("button", { name: "Add" }).click();

  return totpPage;
};
