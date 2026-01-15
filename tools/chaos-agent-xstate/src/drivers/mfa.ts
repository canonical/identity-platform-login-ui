import { BaseDriver } from './base';
import chalk from 'chalk';
import * as OTPAuth from 'otpauth';
import * as fs from 'fs';
import * as path from 'path';

export class MfaDriver extends BaseDriver {

    async skipTotpSetup() {
        this.log(chalk.yellow('Skipping TOTP setup -> Forced navigation to WebAuthn setup'));
        const currentUrl = this.page.url();
        const newUrl = currentUrl.replace('setup_secure', 'setup_passkey');
        await this.page.goto(newUrl);
        await this.page.waitForLoadState('networkidle');
    }

    async setupTotp() {
        this.log(chalk.blue('Setting up TOTP...'));

        // 1. Find the Secret Key
        let secret = '';
        try {
            // Try standard Kratos/Ory UI elements
            const secretElement = this.page.locator('div[data-testid="node/text/todp_secret_key/text"]');
            if (await secretElement.isVisible()) {
                secret = (await secretElement.innerText()).replace('Secret: ', '').trim();
            } else {
                // Fallback: finding text "Secret: " in the body
                const body = await this.page.innerText('body');
                const match = body.match(/Secret:\s*([A-Z2-7]+)/i);
                if (match) {
                    secret = match[1];
                }
            }
        } catch (e) {
            this.log(chalk.red('Failed to locate TOTP secret key'));
            throw e;
        }

        if (!secret) {
            this.log(chalk.red('TOTP Secret key not found in DOM!'));
            try {
                this.log('Page content: ' + (await this.page.innerHTML('body')));
            } catch {}
            throw new Error('TOTP Secret not found');
        }

        this.state.savedTotpSecret = secret.replace(/\s/g, ''); // Remove spaces
        this.log(chalk.green(`Found TOTP Secret: ${this.state.savedTotpSecret.substring(0, 4)}...`));

        // 2. Generate Code
        const totp = new OTPAuth.TOTP({
            algorithm: 'SHA1',
            digits: 6,
            period: 30,
            secret: OTPAuth.Secret.fromBase32(this.state.savedTotpSecret)
        });
        const code = totp.generate();
        this.log(chalk.blue(`Generated Code: ${code}`));

        // 3. Enter Code
        await this.page.fill('input[name="totp_code"], input[name="code"]', code);
        await this.page.click('button[type="submit"], button:has-text("Save"), button:has-text("Next")');

        // 4. Verify Success
        try {
            await Promise.race([
                this.page.waitForSelector('button:has-text("Delete")', { timeout: 5000 }), // If stays on management
                this.page.waitForURL(/setup_passkey|manage|dashboard/, { timeout: 5000 })
            ]);
            this.log(chalk.green('TOTP Setup complete!'));
        } catch (e) {
            this.log(chalk.yellow('Warning: No clear success signal after TOTP submit'));
        }
    }

    async verifyTotp() {
        this.log(chalk.blue('Verifying TOTP...'));

        if (!this.state.savedTotpSecret) {
            // Try to load from persistent state file if we have a user
            if (this.state.currentUserEmail) {
                try {
                    const statePath = path.resolve('.chaos-agent-state.json');
                    if (fs.existsSync(statePath)) {
                        const fileState = JSON.parse(fs.readFileSync(statePath, 'utf8'));
                        const userState = fileState.users?.[this.state.currentUserEmail];
                        if (userState && userState.totpSecret) {
                            this.state.savedTotpSecret = userState.totpSecret;
                            this.log(chalk.green(`Loaded TOTP secret from file for ${this.state.currentUserEmail}`));
                        }
                    }
                } catch (e) {
                    this.log(chalk.yellow(`Failed to load persistent state: ${e}`));
                }
            }
        }

        if (!this.state.savedTotpSecret) {
            throw new Error('No saved TOTP secret! Setup must run before Verify.');
        }

        const totp = new OTPAuth.TOTP({
            algorithm: 'SHA1',
            digits: 6,
            period: 30,
            secret: OTPAuth.Secret.fromBase32(this.state.savedTotpSecret)
        });
        const code = totp.generate();
        this.log(chalk.blue(`Generated Code: ${code}`));

        await this.page.fill('input[name="totp_code"], input[name="code"]', code);
        await this.page.click('button[type="submit"], button:has-text("Sign in")');

        // Robust wait for success
        try {
            await Promise.race([
                this.page.waitForURL(/dashboard|welcome|settings/, { timeout: 10000 }),
                this.page.waitForSelector('a:has-text("Sign out"), button:has-text("Sign out")', { timeout: 10000 })
            ]);
        } catch (e) {
            this.log(chalk.yellow(`Warning: Navigation after TOTP verify timed out, but proceeding... URL: ${this.page.url()}`));
        }

        await this.page.waitForLoadState('networkidle');
        this.log(chalk.green('TOTP Submitted'));
    }

    async enterBackupCode(code: string) {
        this.log(chalk.blue('Entering Backup Code...'));
        await this.page.waitForLoadState('networkidle');

        const btn = this.page.getByRole('button', { name: 'Use backup code instead' });
        try {
            if (await btn.isVisible()) {
                this.log(chalk.gray('Clicking "Use backup code instead"...'));
                await btn.click();
            }
        } catch (e) {
             // Ignore if button disappears or detached
        }

        // Use label as per spec, fallback to known names
        try {
            await this.page.getByLabel("Backup recovery code").fill(code);
        } catch (e) {
            this.log(chalk.yellow('Could not find input by label, trying selector...'));
            await this.page.fill('input[name="lookup_secret"], input[name="backup_code"]', code);
        }

        await this.page.click('button[type="submit"], button:has-text("Sign in")');
    }

    async enableWebAuthn() {
        if (!this.state.cdpSession) {
            this.log(chalk.blue('Creating new CDP Session'));
            this.state.cdpSession = await this.page.context().newCDPSession(this.page);
            await this.state.cdpSession.send('WebAuthn.enable');
        }

        // Strategy: Reuse existing authenticator to preserve private keys and sign counts.
        if (this.state.authenticatorId) {
            this.log(chalk.gray(`Reusing active Virtual Authenticator: ${this.state.authenticatorId}`));
            // Sanity check: Does it still exist?
            try {
                const check = await this.state.cdpSession!.send('WebAuthn.getCredentials', {
                    authenticatorId: this.state.authenticatorId
                });
                // this.log(chalk.gray(`Authenticator has ${check.credentials.length} credentials.`));
            } catch(e) {
                this.log(chalk.red('Authenticator check failed! It might have been detached. Creating new one.'));
                this.state.authenticatorId = null;
                // fall through to create new
            }
        }

        if (this.state.authenticatorId) return;

        this.log(chalk.blue('Adding Virtual Authenticator'));
        const result = await this.state.cdpSession!.send('WebAuthn.addVirtualAuthenticator', {
            options: {
                protocol: 'ctap2',
                transport: 'usb',
                hasResidentKey: true,
                hasUserVerification: true,
                isUserVerified: true,
                automaticPresenceSimulation: true
            }
        });
        this.state.authenticatorId = result.authenticatorId;

        // We DO NOT restore credentials here because we can't restore private keys.
        if (this.state.savedCredential) {
             this.log(chalk.yellow('Warning: Created new authenticator but have saved credentials. Verify will likely fail due to key mismatch.'));
        }
    }

    async registerWebAuthn() {
        await this.enableWebAuthn();
        this.log(chalk.blue('Registering new WebAuthn credential'));

        // Fill the name
        try {
            const nameInput = this.page.getByLabel('Security key name');
            if (await nameInput.isVisible()) {
                 const keyName = `XState Key ${Date.now()}`;
                 this.log(chalk.gray(`Naming key: ${keyName}`));
                 await nameInput.fill(keyName);
            }
        } catch (e) {
            this.log(chalk.yellow('Could not find name input, trying to proceed anyway'));
        }

        // Handle "Add security key"
        await this.page.click('button:has-text("Add security key")');

        // Wait for success
        try {
            await Promise.race([
                this.page.waitForSelector('button:has-text("Delete")', { timeout: 5000 }),
                this.page.waitForURL(/manage|dashboard|welcome/, { timeout: 5000 })
            ]);
            console.log(chalk.green('WebAuthn registration action completed (UI updated or navigated)'));
        } catch (e) {
            console.log(chalk.yellow('Warning: UI did not update as expected, but checking for credentials anyway'));
        }

        // EXPORT CREDENTIAL (Always try this)
        console.log(chalk.blue('Exporting generated credential...'));
        try {
            const response = await this.state.cdpSession!.send('WebAuthn.getCredentials', {
                authenticatorId: this.state.authenticatorId!
            });
            if (response.credentials && response.credentials.length > 0) {
                this.state.savedCredential = response.credentials[0];
                console.log(chalk.green(`Credential saved for re-use! (ID: ${this.state.savedCredential.credentialId.substring(0, 10)}...)`));
            } else {
                console.log(chalk.red('No credentials found in authenticator! Registration likely failed.'));
            }
        } catch(e) {
            console.log(chalk.red('Failed to export credentials: ' + e));
        }
    }

    async performWebAuthnLogin() {
        await this.enableWebAuthn(); // Ensure session is active

        // Ensure the page is settled and cookies are applied
        this.log(chalk.blue('Reloading page to ensure session cookies are active...'));
        await this.page.reload({ waitUntil: 'networkidle' });

        this.log(chalk.blue('Clicking "Sign in with Security key"'));

        // Try multiple selectors
        const btn = this.page.locator('button:has-text("Sign in with Security key")')
            .or(this.page.locator('button:has-text("Use security key")'))
            .or(this.page.locator('button:has-text("Sign in with hardware key")'));

        // Brief wait to see if automatic presence simulation triggers navigation
        try {
            await this.page.waitForURL(/manage|setup/, { timeout: 2000 });
            this.log(chalk.green('Automatic navigation detected!'));
            return;
        } catch {
            // Not navigated yet, proceed to click
        }

        await btn.first().click({ timeout: 5000 });

        // Wait for redirection to dashboard/management page
        this.log(chalk.blue('Waiting for redirection to dashboard...'));
        try {
            // Allow /manage or /setup (often setup_secure if MFA incomplete)
            await this.page.waitForURL(/manage|setup/, { timeout: 10000 });
            this.log(chalk.green('Successfully authenticated with WebAuthn!'));
        } catch (e) {
            this.log(chalk.red('Failed to reach dashboard. Current URL: ' + this.page.url()));
            throw e;
        }
    }
}
