import { Page, expect } from "@playwright/test";

export const startNewAuthFlow = async (page: Page) => {
  const url = "http://127.0.0.1:4446/";
  await page.goto(url);
  await page.getByRole("link", { name: "Authorize application" }).click();
  await expect(page.getByText("Sign in to OIDC App")).toBeVisible();
};

export const finishAuthFlow = async (page: Page) => {
  const redirectURI = "http://127.0.0.1:4446/callback";
  await page.waitForURL(redirectURI + "?*");

  const data = await page.content();
  expect(data).toContain("Access Token");
  expect(data).toContain("Refresh Token");
  expect(data).toContain("ID Token");
};
