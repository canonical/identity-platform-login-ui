import { test, expect } from '@playwright/test';
import { KratosPage, EmailService } from '../library';

const TEST_EMAIL = 'recovery-test@example.com';
const ORIGINAL_PASSWORD = 'OriginalPassword123!';
const NEW_PASSWORD = 'NewPassword123!';
const BACKUP_CODE = 'backup-code-12345'; // Replace with actual backup code if testing 2FA

test.describe('Self-service account recovery - Happy Path', () => {
    test('Complete recovery flow with chaos check and 2FA', async ({ page }) => {
        const kratos = new KratosPage(page);
        const emailService = new EmailService();

        // Prerequisites: Ensure account exists
        // Note: In a real test, you would create the account here or in beforeEach

        // Step 1: Navigate to the Login Page
        await kratos.gotoLogin();
        await expect(page).toHaveTitle(/Sign in/);

        // Step 2: Click on "Reset password"
        await page.getByText('Forgot password').click();
        await expect(page).toHaveURL(/.*reset_email.*/);

        // Step 3: Chaos Check - Navigate Back to Login, then Forward to Reset Password
        console.log('[Chaos Check] Testing browser navigation...');
        await page.goBack();
        await expect(page).toHaveTitle(/Sign in/);
        await expect(page.getByLabel('Email')).toBeVisible();

        await page.goForward();
        await expect(page).toHaveURL(/.*reset_email.*/);
        await expect(page.getByLabel('Email')).toBeVisible();
        console.log('[Chaos Check] ✅ Navigation test passed');

        // Step 4: Enter an email associated with an existing account
        const initialEmailCount = await emailService.getEmailCount(TEST_EMAIL);
        console.log(`[Recovery] Initial email count: ${initialEmailCount}`);
        await kratos.submitRecoveryEmail(TEST_EMAIL);

        // Step 5: Click on "Submit" (handled by submitRecoveryEmail)
        // Wait for code input to appear
        await expect(page.getByLabel('Code')).toBeVisible({ timeout: 10000 });
        console.log('[Recovery] Code input visible, waiting for email...');

        // Step 6: Use EmailService to fetch the recovery code from Mailslurper
        const recoveryCode = await emailService.getLatestCode(TEST_EMAIL, 15, 2000, initialEmailCount);
        expect(recoveryCode).toBeTruthy();
        console.log(`[Recovery] ✅ Retrieved recovery code: ${recoveryCode}`);

        // Step 7: Insert the recovery code
        await kratos.submitRecoveryCode(recoveryCode!);
        console.log('[Recovery] Recovery code submitted');

        // Step 8: Complete 2FA if enforce_mfa is enabled
        const requires2FA = await kratos.is2FARequired();
        if (requires2FA) {
            console.log('[2FA] 2FA challenge detected');

            // Try TOTP first, fall back to backup code
            const totpInput = page.getByLabel('Authentication code');
            const backupInput = page.getByLabel('Backup code');

            if (await totpInput.isVisible()) {
                console.log('[2FA] TOTP required - Note: This test requires manual TOTP or mock');
                // In production test, you would generate TOTP from shared secret
                // For now, use backup code if available
                const useBackupLink = page.getByText('Use backup code');
                if (await useBackupLink.isVisible()) {
                    await useBackupLink.click();
                    await kratos.submitBackupCode(BACKUP_CODE);
                    console.log('[2FA] ✅ Backup code submitted');
                }
            } else if (await backupInput.isVisible()) {
                await kratos.submitBackupCode(BACKUP_CODE);
                console.log('[2FA] ✅ Backup code submitted');
            }
        } else {
            console.log('[2FA] No 2FA required');
        }

        // Step 9: Type a new password
        await expect(page).toHaveURL(/.*reset_password.*/, { timeout: 10000 });
        await expect(page.getByLabel('Password')).toBeVisible();
        await kratos.submitNewPassword(NEW_PASSWORD);
        console.log('[Recovery] New password submitted');

        // Expected Result: User is logged in and redirected to TOTP settings
        // The "password was changed successfully" message is displayed
        await expect(page.getByText(/password.*changed.*successfully/i)).toBeVisible({ timeout: 10000 });
        console.log('[Recovery] ✅ Success message displayed');

        // Verify redirect to settings or dashboard
        await expect(page).toHaveURL(/.*settings.*|.*dashboard.*|.*totp.*/, { timeout: 5000 });
        console.log('[Recovery] ✅ Redirected to user area');
    });
});
