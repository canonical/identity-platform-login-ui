import { test, expect, Page } from "@playwright/test";
import { enterTotpCode, setupTotp } from "./helpers/totp";
import { finishAuthFlow, startNewAuthFlow } from "./helpers/oidc_client";
import { resetIdentities } from "./helpers/kratosIdentities";
import { USER_EMAIL, userPassLogin } from "./helpers/login";

// Goes back to a page in the browser history, however many entries away.
const backTo = async (page: Page, url: string) => {
  const cdp = await page.context().newCDPSession(page);
  const { currentIndex, entries } = await cdp.send("Page.getNavigationHistory");
  const entry = entries
    .slice(0, currentIndex)
    .reverse()
    .find((e) => e.url === url);
  if (!entry) {
    throw new Error(`${url} not found in the browser history`);
  }
  await cdp.send("Page.navigateToHistoryEntry", { entryId: entry.id });
  await expect(page).toHaveURL(url);
};

const expectAccountPage = async (page: Page) => {
  await expect(page).toHaveURL(/\/ui\/manage_details/);
  await expect(page.getByLabel("Email address")).toHaveValue(USER_EMAIL);
};

test("code submitted on the second factor page of a completed login", async ({
  browser,
  page,
}) => {
  resetIdentities();
  await startNewAuthFlow(page);
  await userPassLogin(page);
  const setupKey = await setupTotp(page);
  await finishAuthFlow(page);

  // Sign in in a new context as user is already authenticated within the current context
  const newContext = await browser.newContext();
  const newPage = await newContext.newPage();

  await newPage.goto("http://localhost/ui/login");
  await userPassLogin(newPage);
  await expect(
    newPage.getByRole("heading", { name: "Verify your identity" }),
  ).toBeVisible();
  const secondFactorUrl = newPage.url();
  await enterTotpCode(newPage, setupKey);
  await expectAccountPage(newPage);

  await backTo(newPage, secondFactorUrl);
  await enterTotpCode(newPage, setupKey);

  // The user is already signed in: they are sent to their account page
  await expectAccountPage(newPage);
});
