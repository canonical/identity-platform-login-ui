const { chromium } = require('playwright');
const { execSync } = require('child_process');
const fs = require('fs');

// Configuration
const EMAIL = 'webauthn-tester@example.com';
const PASSWORD = 'Password123!';
const ADMIN_API = 'http://localhost:4434';
const UI_BASE = 'http://localhost';

async function main() {
  console.log('🚀 Starting WebAuthn Registration Script...');

  // 1. Reset/Create Identity
  console.log('👤 Preparing user identity...');
  try {
    // Get ID if exists
    const listCmd = `curl --silent "${ADMIN_API}/admin/identities"`;
    const identities = JSON.parse(execSync(listCmd).toString());
    const existing = identities.find(i => i.traits.email === EMAIL);

    if (existing) {
      console.log(`   - Deleting existing user ${existing.id}...`);
      execSync(`curl --silent -X DELETE "${ADMIN_API}/admin/identities/${existing.id}"`);
    }

    // Create new
    console.log('   - Creating new user...');

    const createPayload = JSON.stringify({
      schema_id: "default",
      traits: { email: EMAIL },
      verifiable_addresses: [{ value: EMAIL, verified: true, via: "email" }],
      credentials: { password: { config: { password: PASSWORD } } }
    });

    // Check if oathtool is available
    try {
      execSync('oathtool --version');
    } catch {
      console.error('❌ oathtool is not installed. Please install it (e.g. sudo apt install oathtool)');
      process.exit(1);
    }

    execSync(`curl --silent -X POST "${ADMIN_API}/admin/identities" -H "Content-Type: application/json" -d '${createPayload}'`);
    console.log('   - User created.');

  } catch (e) {
    console.error('❌ Failed to manage identity:', e.message);
    process.exit(1);
  }

  // 2. Launch Browser with Virtual Authenticator
  console.log('🌐 Launching browser...');
  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext();
  const page = await context.newPage();

  // 3. Initialize CDP for WebAuthn
  console.log('🔑 Initializing Virtual Authenticator...');
  const cdp = await context.newCDPSession(page);
  await cdp.send('WebAuthn.enable');
  const { authenticatorId } = await cdp.send('WebAuthn.addVirtualAuthenticator', {
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
    // 4. Login
    console.log('🔐 Logging in...');
    await page.goto(`${UI_BASE}/ui/manage_passkey`);
    // wait to be redirected to /ui/login
    await page.waitForURL(`${UI_BASE}/ui/login**`);
    await page.waitForLoadState('networkidle');
    await page.getByLabel('Email').fill(EMAIL);
    await page.getByLabel('Password').fill(PASSWORD);
    await page.getByRole('button', { name: 'Sign in', exact: true }).click();

    // wait for redirection after login
    await page.waitForURL(`${UI_BASE}/ui/setup_**`);

    // 5. Navigate to Security Keys
    console.log('🗝️  Registering Security Key...');
    await page.goto(`${UI_BASE}/ui/manage_passkey`);
    await page.waitForLoadState('networkidle');

    await page.getByLabel('Security key name').fill('Virtual YubiKey');
    await page.getByRole('button', { name: 'Add security key' }).click();

    // 6. Verification
    console.log('✅ Waiting for registration confirmation...');
    // The UI should show the new key in a list.
    await page.waitForSelector('text="Virtual YubiKey"', { timeout: 10000 });

    console.log('🎉 SUCCESS: WebAuthn Key Registered!');
    console.log('---------------------------------------------------');
    console.log(`User: ${EMAIL}`);
    console.log(`Pass: ${PASSWORD}`);

    // store the webauthn key info from cdp to a file for login script to use later
    const passkey = await cdp.send('WebAuthn.getCredentials', {
      authenticatorId,
    })
    fs.writeFileSync('webauthn-authenticators.json', JSON.stringify(passkey, null, 2));
    console.log('🔐 WebAuthn authenticators info saved to webauthn-authenticators.json');
  } catch (error) {
    console.error('❌ An error occurred during the flow:', error);
    await page.screenshot({ path: 'error-state.png' });
    console.error('📸 Screenshot saved to error-state.png');
  } finally {
    await browser.close();
  }
}

main();
