import { Graph } from '../../core/graph';
import { Page } from 'playwright';
import { TOTPService } from '../../core/services/totp';
import { agentState } from '../../core/state';
import { UserProfile } from '../../core/user-profiles';

export function loginRules(graph: Graph) {
    // 1. Define States
    // TOTP Verification State (check this FIRST - more specific)
    graph.addState({
        id: 'totp-verify',
        urlPattern: /.*\/ui\/login.*/,
        name: 'TOTP Verification',
        contentCheck: async (page: Page) => {
            // Check for explicit TOTP input fields first
            const hasAuthCode = await page.locator('input[autocomplete="one-time-code"]').isVisible().catch(() => false);
            const hasCodeInput = await page.locator('input[name="code"]').isVisible().catch(() => false);
            const hasTotpCode = await page.locator('input[name="totp_code"]').isVisible().catch(() => false);

            if (hasAuthCode || hasCodeInput || hasTotpCode) {
                return true;
            }

            // Check for "Verify your identity" title BUT ensure it's not a pure WebAuthn page
            // If it's a pure WebAuthn page, we delay matching until webauthn rules ran.
            const title = await page.title();
            const isVerifyPage = title.includes('Verify your identity') || title.includes('Verify');

            if (isVerifyPage) {
                // If it is verify page but NO input code, we assume it's NOT TOTP
                // (e.g. it might be WebAuthn only, or passwordless)
                return false;
            }

            return false;
        }
    });

    // Login Page (initial state - less specific, checked second)
    graph.addState({
        id: 'login',
        urlPattern: /.*\/ui\/login.*/,
        name: 'Login Page',
        contentCheck: async (page: Page) => {
            // Check page title
            const title = await page.title();

            // Login page has title like "Sign in" or "Sign in to OIDC App"
            // NOT "Verify your identity"
            const isSignInPage = title.includes('Sign in') && !title.includes('Verify');

            if (!isSignInPage) {
                return false;
            }

            // Double-check: should have email/password fields
            const hasEmail = (await page.locator('input[name="identifier"]').count()) > 0;
            const hasPassword = (await page.locator('input[type="password"]').count()) > 0;

            return hasEmail && hasPassword;
        }
    });

    // Note: 'dashboard' state moved to dashboard.ts

    // 2. Define Transitions
    // Login -> TOTP Verification (password login)
    graph.addTransition({
        from: 'login',
        to: 'totp-verify',
        trigger: {
            id: 'login-password-submit',
            description: 'Submit email and password',
            condition: async (page: Page) => {
                // Check for identifier and password fields that are ENABLED
                const identifierInput = page.locator('input[name="identifier"]');
                const passwordInput = page.locator('input[type="password"]');

                const hasIdentifier = await identifierInput.isVisible().catch(() => false);
                const hasPassword = await passwordInput.isVisible().catch(() => false);

                if (!hasIdentifier || !hasPassword) {
                    return false;
                }

                // Check if identifier is enabled (not disabled during TOTP wait)
                const isIdentifierEnabled = await identifierInput.isEnabled().catch(() => false);

                return hasIdentifier && hasPassword && isIdentifierEnabled;
            },
            action: async (page: Page) => {
                console.log('[Login] Submitting credentials...');

                // Get user profile from state, fallback to default
                const userProfile = agentState.get<UserProfile>('userProfile');
                const email = userProfile?.email || 'webauthn-tester@example.com';
                const password = userProfile?.password || 'Password123!';

                console.log(`[Login] Using credentials: ${email}`);

                // Fill identifier
                if (await page.locator('input[name="identifier"]').isVisible()) {
                    await page.fill('input[name="identifier"]', email);
                }

                // Fill password
                await page.fill('input[type="password"]', password);
                await page.click('button[type="submit"], button:has-text("Sign in")');
                await page.waitForLoadState('networkidle');
            }
        }
    });

    // TOTP Verify -> Setup/Dashboard (submit TOTP code)
    graph.addTransition({
        from: 'totp-verify',
        to: 'setup-secure', // Could also go to dashboard if already set up
        trigger: {
            id: 'totp-verify-submit',
            description: 'Submit TOTP verification code',
            condition: async (page: Page) => {
                const hasTotpCode = await page.getByLabel('Authentication code', { exact: false }).isVisible().catch(() => false);
                return hasTotpCode;
            },
            action: async (page: Page) => {
                console.log('[Login] TOTP verification page detected, generating code...');

                // Check for existing error message before submitting
                const errorVisible = await page.getByText('The provided authentication code is invalid').isVisible().catch(() => false);
                if (errorVisible) {
                    console.log('[Login] Detected invalid code error. Waiting for new TOTP window (5s)...');
                    await page.waitForTimeout(5000);
                }

                // Check if the input is already disabled (submission in progress)
                const totpInput = page.getByLabel('Authentication code', { exact: false });

                try {
                    const isDisabled = await totpInput.isDisabled({ timeout: 5000 });

                    if (isDisabled) {
                        console.log('[Login] TOTP input is disabled, submission already in progress');
                        // Wait for URL to change or page to update
                        await page.waitForURL(/.*/, { waitUntil: 'commit', timeout: 5000 }).catch(() => {});
                        return;
                    }
                } catch (error) {
                    // Element might have disappeared (successful login), just return
                    console.log('[Login] TOTP input no longer available, login likely succeeded');
                    return;
                }

                // Try to retrieve the TOTP secret from current user profile
                const userProfile = agentState.get<UserProfile>('userProfile');
                let secret: string | undefined;

                if (userProfile && userProfile.totpSecret) {
                    secret = userProfile.totpSecret;
                    console.log(`[Login] Using TOTP secret from user profile (${userProfile.email})`);
                }

                // Legacy fallback
                if (!secret) {
                    secret = agentState.get<string>('totpSecret');
                }

                // Fallback to environment variable or default test secret
                if (!secret) {
                    secret = process.env.TOTP_SECRET || 'JBSWY3DPEHPK3PXP';
                    console.log('[Login] Using fallback TOTP secret (not from setup phase)');
                } else {
                    console.log('[Login] Using TOTP secret from setup phase');
                }

                const code = TOTPService.generate(secret);

                // Anti-Loop: Check if we just submitted this code
                const lastCode = agentState.get<string>('lastTotpCode');
                if (lastCode && lastCode === code) {
                    console.log(`[Login] Code ${code} already submitted recently. Waiting for next TOTP window...`);
                    // Wait enough time for a new code to be valid (usually 30s period, but we just wait a bit)
                    await page.waitForTimeout(5000);
                    return;
                }

                console.log(`[Login] Generated TOTP code: ${code}`);
                agentState.set('lastTotpCode', code);

                await totpInput.fill(code);
                await page.getByRole('button', { name: 'Sign in' }).click();

                console.log('[Login] Waiting for navigation after TOTP submit...');
                // Wait for URL to change (navigation after successful TOTP)
                await page.waitForURL(/.*/, { waitUntil: 'commit', timeout: 10000 }).catch(() => {
                    console.log('[Login] No URL change detected after TOTP submit');
                });

                const newUrl = page.url();
                console.log(`[Login] URL after TOTP login: ${newUrl}`);
            }
        }
    });
}
