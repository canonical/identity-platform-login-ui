import { BrowserContext } from "playwright-core";

const MAIL_SLURP_URL = "http://localhost:4436";

export const getRecoveryCodeFromMailSlurp = async (context: BrowserContext) => {
  const mailSlurp = await context.newPage();
  await mailSlurp.waitForTimeout(1000); // wait for email to be sent
  await mailSlurp.goto(MAIL_SLURP_URL);
  await mailSlurp
    .getByRole("link", { name: "Recover access to your account" })
    .first()
    .click();
  const text = await mailSlurp.locator("#mailDetails").textContent();
  const cleanText = text?.replace(/\n+/g, " ").replace("/  +/g", " ");
  const code = cleanText?.match(/code: (\d+)/)?.[1];
  if (!code) {
    throw new Error("Code not found");
  }

  return code;
};
