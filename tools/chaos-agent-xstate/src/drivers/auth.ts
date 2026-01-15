import { BaseDriver } from './base';
import chalk from 'chalk';

export class AuthDriver extends BaseDriver {

    async navigateToLogin(url: string) {
        this.log(chalk.blue(`Navigating to ${url}`));
        await this.page.goto(url);
        await this.page.waitForLoadState('networkidle');
    }

    async performLogin({ email, password }: { email: string, password?: string }) {
        this.log(chalk.blue(`Logging in as ${email}`));
        this.state.currentUserEmail = email; // Update Shared State
        this.log(chalk.gray(`Current URL: ${this.page.url()}`));

        // Note: We do NOT enable WebAuthn here to avoid accidental usage/counter increment

        try {
            await this.page.fill('input[name="identifier"]', email, { timeout: 10000 });
        } catch (e) {
             this.log(chalk.red('Login input not found. Page content:'));
             try {
                this.log((await this.page.innerHTML('body')).substring(0, 500));
             } catch {}
             throw e;
        }

        // Handle Identifier First (Split Flow) vs Combined Flow
        // Check if password field is visible
        const passwordInput = this.page.locator('input[type="password"]');

        if (await passwordInput.isVisible()) {
            // Combined flow or Password step already active

            if (password) {
                await passwordInput.fill(password);
                await this.page.click('button[type="submit"], button:has-text("Sign in")');
            }
        } else {
            this.log(chalk.gray('Password argument provided but input not visible. Assuming Identifier First flow.'));

            // Check for security key button on first page (just in case)
            const secBtn1 = this.page.locator('button:has-text("Sign in with Security key")').or(this.page.locator('button:has-text("Use security key")'));
            if (email.includes('webauthn-dynamic') && await secBtn1.isVisible()) {
                 this.log(chalk.blue('Detected Security Key option on first page - attempting Passwordless flow'));
                 return;
            }

            await this.page.click('button[type="submit"], button:has-text("Sign in"), button:has-text("Next")');

            // Wait for either Password Input OR Security Key Button
            try {
                await Promise.race([
                    passwordInput.waitFor({ state: 'visible', timeout: 5000 }),
                    // secBtn1.waitFor({ state: 'visible', timeout: 5000 }) // Disabled passwordless check
                ]);
            } catch (e) {
                 this.log(chalk.yellow('Waiting for next step timed out.'));
            }

            if (password && await passwordInput.isVisible()) {
                 await passwordInput.fill(password);
                 await this.page.click('button[type="submit"], button:has-text("Sign in")');
            }
        }

        await this.page.waitForTimeout(2000); // Wait for transition

        // DEBUG: Check for login success
        const cookies = await this.page.context().cookies();
        const sessionCookie = cookies.find(c => c.name.includes('session'));
        this.log(chalk.yellow(`DEBUG: Cookies after login: ${JSON.stringify(cookies.map(c => c.name))}`));
        if (sessionCookie) {
            this.log(chalk.green('DEBUG: Session cookie found!'));
        } else {
            this.log(chalk.red('DEBUG: NO SESSION COOKIE FOUND after login attempt!'));
        }
    }

    async performLogout() {
        this.log(chalk.blue('Logging out...'));
        await this.page.waitForLoadState('domcontentloaded');

        // Clear all persistent state
        await this.page.context().clearCookies();
        try {
            await this.page.evaluate(() => {
                localStorage.clear();
                sessionStorage.clear();
            });
        } catch (e) {
            // Ignore if context is restricted
        }
        await this.page.waitForTimeout(500);

        // Explicitly navigate to login page with a cache buster
        const currentUrl = this.page.url();
        const baseUrl = currentUrl.includes('/ui')
            ? currentUrl.split('/ui')[0]
            : (currentUrl.includes('localhost') ? 'http://localhost' : new URL(currentUrl).origin);

        const loginUrl = `${baseUrl}/ui/login?force_relogin=${Date.now()}`;
        this.log(chalk.blue(`Navigating to login page: ${loginUrl}`));

        await this.page.goto(loginUrl);
        await this.page.waitForURL(/login/);
        await this.page.waitForLoadState('networkidle');
    }

    async analyzeCurrentState(): Promise<string> {
        const url = this.page.url();
        this.log(chalk.gray(`Analyzing URL: ${url}`));

        if (url.includes('setup_secure')) return 'setup_totp';
        if (url.includes('setup_passkey') || url.includes('manage_passkey')) return 'setup_passkey';
        if (url.includes('manage')) return 'dashboard'; // Loose match for dashboard
        if (url.includes('login')) {
            // Check if it's password login or WebAuthn verify
            const hasWebAuthnBtn = await this.page.locator('button:has-text("Sign in with Security key")')
                .or(this.page.locator('button:has-text("Use security key")'))
                .or(this.page.locator('button:has-text("Sign in with hardware key")'))
                .isVisible().catch(() => false);

            this.log(`[Analyze] Has WebAuthn Button: ${hasWebAuthnBtn}`);
             if (hasWebAuthnBtn) return 'webauthn_verify';

             // Check if it is a TOTP verification step
             const hasTotpInput = await this.page.locator('input[name="totp_code"], input[name="code"]')
                .isVisible().catch(() => false);
             this.log(`[Analyze] Has TOTP Input: ${hasTotpInput}`);
             if (hasTotpInput) return 'totp_verify';

             // Check if we are actually just logged in but stuck on login page
             if (await this.page.getByText('You are already logged in').isVisible().catch(() => false)) {
                 this.log('[Analyze] "You are already logged in" detected -> dashboard');
                 return 'dashboard';
             }

             return 'login';
        }

        return 'unknown';
    }

    async validateSession() {
        this.log(chalk.blue('Validating active session...'));
        try {
             // We expect to be on a management or setup page
             await this.page.waitForURL(/manage|setup/, { timeout: 10000 });

             // Check for Side Navigation links which appear only when authenticated
             await this.page.waitForSelector('a:has-text("Personal details")', { timeout: 5000 });

             this.log(chalk.green('Session validated: Management navigation visible.'));
        } catch (e) {
             this.log(chalk.red('Session validation failed!'));
             throw e;
        }
    }
}
