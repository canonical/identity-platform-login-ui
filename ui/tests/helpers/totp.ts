import { expect, Page } from "@playwright/test";
import { execSync } from "child_process";

export const enterTotpCode = async (page: Page, setupKey: string) => {
  await expect(page.getByText("Verify your identity")).toBeVisible();
  await expect(page).toHaveScreenshot({ fullPage: true, maxDiffPixels: 500 });
  const code = getTotpCode(setupKey);
  await page.getByLabel("Authentication code").fill(code);
  await page.getByRole("button", { name: "Sign in" }).click();
};

const getTotpCode = (setupKey: string) => {
  return execSync(`oathtool -b --totp '${setupKey}'`).toString();
};

export const setupTotp = async (page: Page) => {
  await expect(page.getByText("Secure your account")).toBeVisible();
  await expect(page).toHaveScreenshot({
    fullPage: true,
    maxDiffPixelRatio: 0.05,
  }); // the code differs every time
  const setupKey = await getTotpSetupKey(page);
  const totpCode = getTotpCode(setupKey);
  await page.getByLabel("Verify code").fill(totpCode);
  await page.getByRole("button", { name: "Save" }).click();
  return setupKey;
};

const getTotpSetupKey = async (page: Page) => {
  const setupKey = await page.locator("pre").textContent();
  if (!setupKey) {
    throw new Error("TOTP setup code not found");
  }
  return setupKey;
};
