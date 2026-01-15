import { Page } from 'playwright';
import chalk from 'chalk';
import { AuthDriver } from './auth';
import { MfaDriver } from './mfa';
import { RecoveryDriver } from './recovery';
import { SettingsDriver } from './settings';
import { DriverState } from './base';

export class Driver {
    public auth: AuthDriver;
    public mfa: MfaDriver;
    public recovery: RecoveryDriver;
    public settings: SettingsDriver;

    // Shared State
    private state: DriverState = {
        currentUserEmail: null,
        cdpSession: null,
        authenticatorId: null,
        savedTotpSecret: null,
        savedCredential: null
    };

    private logger: (msg: string) => void;

    constructor(private page: Page, logger?: (msg: string) => void) {
        this.logger = logger || console.log;

        // Initialize sub-drivers with shared state
        this.auth = new AuthDriver(page, this.state, this.logger);
        this.mfa = new MfaDriver(page, this.state, this.logger);
        this.recovery = new RecoveryDriver(page, this.state, this.logger);
        this.settings = new SettingsDriver(page, this.state, this.logger);

        // Global Page Listeners
        this.page.on('console', msg => {
            if (msg.type() === 'error') this.logger(chalk.red(`BROWSER ERROR: ${msg.text()}`));
        });
        this.page.on('pageerror', err => {
            this.logger(chalk.red(`BROWSER EXCEPTION: ${err.message}`));
        });

        this.page.on('response', async response => {
             if (response.status() === 500) {
                 this.logger(chalk.red(`BROWSER 500 ERROR: ${response.url()}`));
                 try {
                     const text = await response.text();
                     this.logger(chalk.red(`Response Body: ${text.substring(0, 1000)}`));
                 } catch (e) {
                     this.logger(chalk.red('Could not read response body'));
                 }
             }
        });
    }

    // Facade/Proxy methods if needed (currently we access driver.auth.* directly)
}
