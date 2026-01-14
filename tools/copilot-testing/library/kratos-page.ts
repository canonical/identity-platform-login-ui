import { Page, expect } from '@playwright/test';

export class KratosPage {
    constructor(private page: Page) {}

    async gotoLogin() {
        await this.page.goto('/ui/login');
        await expect(this.page).toHaveTitle(/Sign in/);
    }

    async submitLogin(email: string) {
        await this.page.getByLabel('Email').fill(email);
        await this.page.getByRole('button', { name: 'Sign in', exact: true }).click();
    }

    async submitPassword(password: string) {
        await this.page.getByLabel('Password').fill(password);
        await this.page.getByRole('button', { name: 'Sign in', exact: true }).click();
    }

    /**
     * Handles the "Identifier First" flow where email and password might be separate
     */
    async login(email: string, password: string) {
        await this.gotoLogin();
        await this.submitLogin(email);

        // Handle case where password is on same page or next page
        const passwordInput = this.page.getByLabel('Password');
        await expect(passwordInput).toBeVisible();
        await passwordInput.fill(password);
        await this.page.getByRole('button', { name: 'Sign in', exact: true }).click();
    }

    async gotoRecovery() {
        await this.page.goto('/ui/login');
        await this.page.getByText('Forgot password').click();
        await expect(this.page).toHaveURL(/.*reset_email.*/);
    }

    async submitRecoveryEmail(email: string) {
        await this.page.getByLabel('Email').fill(email);
        await this.page.getByRole('button', { name: 'Submit' }).click();
    }

    async submitRecoveryCode(code: string) {
        await this.page.getByLabel('Code').fill(code);
        await this.page.getByRole('button', { name: 'Submit' }).click();
    }

    async submitNewPassword(password: string) {
        await this.page.getByLabel('Password').fill(password);
        await this.page.getByRole('button', { name: 'Submit' }).click();
    }

    async submitTOTP(code: string) {
        await this.page.getByLabel('Authentication code').fill(code);
        await this.page.getByRole('button', { name: 'Submit' }).click();
    }

    async submitBackupCode(code: string) {
        await this.page.getByLabel('Backup code').fill(code);
        await this.page.getByRole('button', { name: 'Submit' }).click();
    }

    /**
     * Check if 2FA is required (TOTP or backup code)
     */
    async is2FARequired(): Promise<boolean> {
        const totpInput = this.page.getByLabel('Authentication code');
        const backupInput = this.page.getByLabel('Backup code');

        try {
            await Promise.race([
                totpInput.waitFor({ state: 'visible', timeout: 2000 }),
                backupInput.waitFor({ state: 'visible', timeout: 2000 })
            ]);
            return true;
        } catch {
            return false;
        }
    }
}
