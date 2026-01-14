#!/usr/bin/env node
/**
 * Remove TOTP authenticator from webauthn-tester user
 * This allows testing WebAuthn-only login when MFA is enforced via AAL
 */

const { chromium } = require('playwright');

const EMAIL = 'webauthn-tester@example.com';
const PASSWORD = 'Password123!';
const TOTP_SECRET = '33YT4SVVEA6UDFER74S4UGEO5ZNAM6SS';
const UI_BASE = process.env.UI_BASE || 'http://localhost';

(async () => {
  console.log('🔧 Removing TOTP from webauthn-tester user...');
  console.log('   This will allow WebAuthn-only login\n');

  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext();
  const page = await context.newPage();

  try {
    // 1. Login with password + TOTP
    console.log('🔐 Logging in...');
    await page.goto(`${UI_BASE}/ui/login`);
    await page.getByLabel('Email').fill(EMAIL);
    await page.getByLabel('Password').fill(PASSWORD);
    await page.getByRole('button', { name: 'Sign in', exact: true }).click();

    // Wait for TOTP verification page
    await page.waitForSelector('input[name="totp_code"]', { timeout: 5000 });
    
    // Generate TOTP code
    const { execSync } = require('child_process');
    const token = execSync(`oathtool -b --totp '${TOTP_SECRET}'`).toString().trim();
    console.log(`   Generated TOTP code: ${token}`);

    // Submit TOTP
    await page.fill('input[name="totp_code"]', token);
    await page.getByRole('button', { name: 'Sign in' }).click();
    
    // Wait for dashboard
    await page.waitForURL(/.*manage_.*/);
    console.log('✓ Logged in successfully\n');

    // 2. Navigate to Authenticator (TOTP) management
    console.log('🗑️  Navigating to Authenticator management...');
    
    // Click on the Authenticator link
    const authenticatorLink = page.locator('a:has-text("Authenticator")');
    await authenticatorLink.click();
    await page.waitForLoadState('networkidle');

    // 3. Check if TOTP is configured and remove it
    const deleteButton = await page.locator('button:has-text("Delete")').first();
    const removeButton = await page.locator('button:has-text("Remove")').first();
    const unlinkButton = await page.locator('button:has-text("Unlink")').first();
    
    const hasDelete = await deleteButton.isVisible().catch(() => false);
    const hasRemove = await removeButton.isVisible().catch(() => false);
    const hasUnlink = await unlinkButton.isVisible().catch(() => false);
    
    console.log(`   Delete button visible: ${hasDelete}`);
    console.log(`   Remove button visible: ${hasRemove}`);
    console.log(`   Unlink button visible: ${hasUnlink}`);
    
    if (!hasDelete && !hasRemove && !hasUnlink) {
      console.log('ℹ️  TOTP is not configured, nothing to remove');
      await browser.close();
      return;
    }

    // 4. Delete TOTP
    console.log('🗑️  Removing TOTP authenticator...');
    if (hasDelete) {
      await deleteButton.click();
    } else if (hasRemove) {
      await removeButton.click();
    } else if (hasUnlink) {
      await unlinkButton.click();
    }
    
    // Wait for deletion to complete
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1000);

    // Verify deletion
    const stillHasTOTP = await page.locator('button:has-text("Delete")').count() > 0;
    
    if (!stillHasTOTP) {
      console.log('✅ TOTP authenticator removed successfully!\n');
      console.log('---------------------------------------------------');
      console.log('User: webauthn-tester@example.com');
      console.log('Authentication: Password + WebAuthn only');
      console.log('TOTP: Removed');
      console.log('---------------------------------------------------');
    } else {
      console.log('⚠️  TOTP might still be configured');
    }

  } catch (error) {
    console.error('❌ Error removing TOTP:', error.message);
    process.exit(1);
  } finally {
    await browser.close();
  }
})();
