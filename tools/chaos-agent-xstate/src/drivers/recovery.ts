import { BaseDriver } from './base';
import { EmailService } from '../utils/email-service';
import chalk from 'chalk';

export class RecoveryDriver extends BaseDriver {
    private emailService: EmailService;
    private initialEmailCount: number = 0;

    constructor(page: any, state: any, logger: any) {
        super(page, state, logger);
        this.emailService = new EmailService(); // Default localhost:4437
    }

    async clickForgotPassword() {
        this.log(chalk.blue('Navigating to Password Recovery...'));
        await this.page.click('a:has-text("Forgot password"), a:has-text("reset password")');
        await this.page.waitForLoadState('networkidle');

        // Validation
        const title = await this.page.title();
        if (title.includes('Enter an email') || (await this.page.locator('input[type="email"]').isVisible())) {
            this.log(chalk.green('Recovery page loaded correctly'));
        } else {
            throw new Error('Failed to load recovery page');
        }
    }

    async submitRecoveryEmail(email: string) {
        this.log(chalk.blue(`Submitting recovery email for: ${email}`));

        // Snapshot email count before action
        this.initialEmailCount = await this.emailService.getEmailCount(email);
        this.log(chalk.gray(`Current email count: ${this.initialEmailCount}`));

        await this.page.locator('input[type="email"]').fill(email);
        await this.page.getByRole('button', { name: 'Reset password' }).click();

        // Wait for code input to appear (indicates form has updated to next step)
        try {
            await this.page.locator('input[name="code"]').waitFor({ state: 'visible', timeout: 10000 });
            this.log(chalk.green('Email submitted, code entry form loaded'));
        } catch (e) {
            this.log(chalk.red('Code input did not appear after submitting email'));
            throw e;
        }
    }

    async enterRecoveryCode(email: string) {
        this.log(chalk.blue('Fetching and entering recovery code...'));

        const code = await this.emailService.getLatestCode(email, 15, 2000, this.initialEmailCount);

        if (!code) {
            throw new Error('Failed to fetch recovery code from email service');
        }

        await this.page.locator('input[name="code"]').fill(code);
        await this.page.getByRole('button', { name: 'Submit' }).click();

        // Wait for Password Reset form
        try {
            await this.page.waitForURL(/.*reset_password.*/, { timeout: 10000 });
            this.log(chalk.green('Code verified, proceeded to password reset page'));
        } catch (e) {
            // Check if we are still on code page with error
            const error = await this.page.locator('.text-error, [data-testid="ui/message/1234"]').textContent().catch(() => null);
            if (error) {
                throw new Error(`Code verification failed: ${error}`);
            }
            throw e;
        }
    }

    async enterNewPassword(newPassword: string) {
        this.log(chalk.blue('Setting new password...'));

        // Fill password
        await this.page.fill('input[name="password"]', newPassword);
        // Trigger blur to ensure validation runs
        await this.page.locator('input[name="password"]').blur();

        // Fill confirmation
        await this.page.fill('input[name="passwordConfirm"]', newPassword);
        // Trigger blur
        await this.page.locator('input[name="passwordConfirm"]').blur();

        await this.page.getByRole('button', { name: 'Reset password' }).click();

        // Success should redirect to login or dashboard
        // If flow ends, we might be at settings or login
        await this.page.waitForLoadState('networkidle');
        this.log(chalk.green('New password submitted'));
    }
}
