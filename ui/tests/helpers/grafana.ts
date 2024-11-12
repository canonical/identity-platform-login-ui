import { Page } from "@playwright/test";

export const startGrafanaNewUserFlow = async (page: Page) => {
  await page.goto("/");
  await page
    .getByRole("link", { name: "Sign in with Identity Platform" })
    .click();
};
