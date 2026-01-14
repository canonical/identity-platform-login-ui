const { chromium } = require('playwright');
const { execSync } = require('child_process');

// Configuration
const EMAIL = 'webauthn-tester@example.com';
const UI_BASE = 'http://localhost';

async function main() {
  console.log('🚀 Starting WebAuthn Login Script...');

  // 1. Launch Browser with Virtual Authenticator
  console.log('🌐 Launching browser...');
  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext();
  const page = await context.newPage();

  // 2. Initialize CDP for WebAuthn
  console.log('🔑 Initializing Virtual Authenticator...');
  const cdp = await context.newCDPSession(page);
  await cdp.send('WebAuthn.enable');
  await cdp.send('WebAuthn.addVirtualAuthenticator', {
    options: {
      protocol: 'ctap2',
      transport: 'usb',
      hasResidentKey: true,
      hasUserVerification: true,
      isUserVerified: true,
      automaticPresenceSimulation: true
    }
  });

  try {
    // 3. Login Page
    console.log('🔐 Navigating to Login...');
    await page.goto(`${UI_BASE}/ui/login`);

    // Verify button exists
    const btnSelector = 'button:has-text("Sign in with Security key")';
    if (!await page.$(btnSelector)) {
        throw new Error("WebAuthn login button not found!");
    }

    // 4. Input Email (Required for identifier-first or to hint which user)
    // Actually, WebAuthn often requires identifying the user first, or it's a passkey flow.
    // In this UI, looking at the HTML:
    // <input name="identifier">
    // <button>Sign in with Security key</button>
    // Kratos typically needs the identifier unless using Resident Keys (Discoverable Credentials).
    // Our registration script set `hasResidentKey: true`.
    // However, the UI might require the identifier field to be filled to know which flow to trigger or to hint.
    // Let's try filling the email first, as that is standard for Kratos identifier-first/MFA.

    console.log('   - Filling identifier...');
    await page.getByLabel('Email').fill(EMAIL);

    console.log('   - Clicking "Sign in with Security key"...');
    // Clicking the button triggers the browser's credential get()
    await page.click(btnSelector);

    // 5. Verification
    console.log('✅ Waiting for successful login...');
    // If successful, we should end up at the dashboard or "Verification" page if MFA is double required?
    // Since we are logging in WITH the key, if it's single factor or if we have 2FA?
    // User `webauthn-tester` has Password + TOTP + WebAuthn.
    // WebAuthn can count as MFA or Passwordless depending on config.
    // If it's used as 2nd factor, we might need password first?
    // The button says "Sign in with Security key", implying passwordless or alternative.
    // Let's see what happens.

    // We expect to see "Canonical SSO" dashboard or similar.
    // Or "Personal details" link in sidebar.
    await page.waitForSelector('text="Personal details"', { timeout: 15000 });

    console.log('🎉 SUCCESS: Logged in with WebAuthn!');
    await page.screenshot({ path: 'login-success.png' });

  } catch (error) {
    console.error('❌ An error occurred during the flow:', error);
    await page.screenshot({ path: 'login-error-state.png' });
    console.log('   - Saved screenshot to login-error-state.png');

    // Debug: Dump page content
    const content = await page.content();
    console.log('   - Page Content length:', content.length);
  } finally {
    await browser.close();
  }
}

main();
