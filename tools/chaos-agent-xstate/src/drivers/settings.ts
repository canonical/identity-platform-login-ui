import { BaseDriver } from './base';
import chalk from 'chalk';

export class SettingsDriver extends BaseDriver {
    async navigateToBackupCodes() {
        this.log(chalk.blue('Navigating to Backup Codes Setup...'));
        // Direct navigation is most robust
        await this.page.goto(this.page.url().replace(/\/ui\/.*$/, '/ui/setup_backup_codes'));
        await this.page.waitForLoadState('networkidle');
    }

    async generateBackupCodes(): Promise<string> {
        this.log(chalk.blue('Generating Backup Codes...'));

        try {
            // Handle existing codes state
            const deactivateBtn = this.page.locator('button:has-text("Deactivate backup codes")');
            if (await deactivateBtn.isVisible()) {
                this.log(chalk.yellow('Backup codes already exist. Deactivating...'));
                await deactivateBtn.click();

                // Handle Confirmation Modal if it appears
                // We assume there's a confirmation since we saw more "Deactivate" buttons in debug
                const confirmBtn = this.page.locator('.p-modal button:has-text("Deactivate backup codes")');
                if (await confirmBtn.isVisible()) {
                     await confirmBtn.click();
                } else {
                     // Check if there is another "Deactivate backup codes" button visible
                     // Sometimes Playwright finds multiple.
                     // Try clicking the last one?
                     const buttons = await this.page.locator('button:has-text("Deactivate backup codes")').all();
                     if (buttons.length > 1) {
                         await buttons[buttons.length - 1].click();
                     }
                }

                await this.page.waitForLoadState('networkidle');
                await this.page.waitForSelector('button:has-text("Create backup codes")');
            }

            await this.page.click('button:has-text("Create backup codes")');
        } catch (e) {
            this.log(chalk.red('Failed to click "Create backup codes". Page content:'));
            // console.log(await this.page.content()); // Too verbose for main log, maybe just title/h1
            const title = await this.page.title();
            const heading = await this.page.locator('h1, h2').first().textContent().catch(() => 'No Heading');
            this.log(`Title: ${title}, Heading: ${heading}`);

            // Check for buttons
            const buttons = await this.page.locator('button').allInnerTexts();
            this.log(`Buttons found: ${buttons.join(', ')}`);
            throw e;
        }

        // Wait for list
        await this.page.waitForSelector('.p-list__item', { timeout: 5000 });

        // Scrape first code
        const code = await this.page.locator('.p-list__item').first().textContent();
        if (!code) {
            throw new Error('Failed to scrape backup code from list');
        }

        this.log(chalk.green(`Captured backup code: ${code}`));
        return code;
    }

    async completeBackupCodesSetup() {
        this.log(chalk.blue('Completing Backup Codes Setup...'));

        await this.page.getByText('I saved the backup codes').click();
        await this.page.click('button:has-text("Create backup codes")'); // It says Create/Save/Done? Reference test says "Create backup codes" again.

        // Verify completion
        await this.page.waitForSelector('text=Account setup complete', { timeout: 10000 });
        this.log(chalk.green('Backup Codes Setup Complete'));
    }
}
