import { Graph } from '../../core/graph';
import { Page } from 'playwright';
import { TOTPService } from '../../core/services/totp';
import { agentState } from '../../core/state';
import { UserProfile } from '../../core/user-profiles';

export function setupRules(graph: Graph) {
    // State: Setup Secure (MFA Setup)
    graph.addState({
        id: 'setup-secure',
        urlPattern: /.*\/ui\/setup_secure.*/,
        name: 'Setup Secure (MFA)'
    });

    // State: Setup Complete (After MFA Setup)
    graph.addState({
        id: 'setup-complete',
        urlPattern: /.*\/ui\/setup_complete.*/,
        name: 'Setup Complete'
    });

    // Skip TOTP and setup WebAuthn (Special case for webauthn-flow)
    graph.addTransition({
        from: 'setup-secure',
        to: 'webauthn-register', // Navigates to passkey setup
        trigger: {
            id: 'skip-totp-webauthn',
            description: 'Skip TOTP and register WebAuthn directly',
            condition: async (page: Page) => {
                const userProfile = agentState.get<UserProfile>('userProfile');
                // Only for webauthn-flow
                if (userProfile?.name !== 'webauthn-flow') return false;

                // Check we are on setup_secure (TOTP page)
                // We assume the default 'setup-totp' rule condition returned false because of the profile check
                // so now we check if we *would* have set up TOTP but are choosing not to.
                const hasTotpInput = await page.locator('input[name="totp_code"]').isVisible().catch(() => false);
                const hasSecretKey = await page.locator('[data-testid="node/text/totp_secret_key/text"]').isVisible().catch(() => false);

                return hasTotpInput && hasSecretKey;
            },
            action: async (page: Page) => {
                console.log('[Setup] Skipping TOTP setup, navigating to Security Key registration...');

                // Construct WebAuthn setup URL (preserving flow/return_to if needed, but simple replacement usually works for UI)
                const currentUrl = page.url();
                const targetUrl = currentUrl.replace('setup_secure', 'setup_passkey');

                console.log(`[Setup] Navigating to: ${targetUrl}`);
                await page.goto(targetUrl);
                await page.waitForLoadState('networkidle');
            }
        }
    });

    // State: Manage Secure (MFA Management - self-serve version of setup_secure)
    graph.addState({
        id: 'manage-secure',
        urlPattern: /.*\/ui\/manage_secure.*/,
        name: 'Manage Secure (MFA Management)'
    });

    // Setup TOTP - Initial setup
    graph.addTransition({
        from: 'setup-secure',
        to: 'setup-secure', // Stay on same state but will trigger different transition next
        trigger: {
            id: 'setup-totp',
            description: 'Setup TOTP Authenticator',
            condition: async (page: Page) => {
                const url = page.url();

                // Skip this rule if we are targeting WebAuthn flow (we want to skip TOTP)
                const userProfile = agentState.get<UserProfile>('userProfile');
                if (userProfile?.name === 'webauthn-flow') {
                    return false;
                }

                console.log(`[Setup] Checking TOTP setup condition on URL: ${url}`);

                // Check for TOTP input and Secret Key presence
                const hasTotpInput = await page.locator('input[name="totp_code"]').isVisible();
                const hasSecretKey = await page.locator('[data-testid="node/text/totp_secret_key/text"]').isVisible();

                console.log(`[Setup] hasTotpInput=${hasTotpInput}, hasSecretKey=${hasSecretKey}`);

                return hasTotpInput && hasSecretKey;
            },
            action: async (page: Page) => {
                console.log('[Setup] Found TOTP Setup. Extracting secret...');

                // Wait for the input to be enabled (not disabled)
                const totpInput = page.locator('input[name="totp_code"]');
                await totpInput.waitFor({ state: 'visible' });

                // Check if already filled or disabled (avoid double submission)
                const isDisabled = await totpInput.isDisabled();
                if (isDisabled) {
                    console.log('[Setup] TOTP input is disabled, submission already in progress');
                    // Wait for page to navigate away
                    await page.waitForURL(/.*/, { waitUntil: 'commit', timeout: 5000 }).catch(() => {});
                    return;
                }

                // Extract secret from the CodeSnippet
                const secretElement = page.locator('[data-testid="node/text/totp_secret_key/text"]');
                const secretText = await secretElement.innerText();
                // Clean up whitespace/labels
                const secret = secretText.replace(/\s/g, '').trim();

                if (!secret) throw new Error('Could not extract TOTP secret');

                // Store the secret for future TOTP verifications
                // refactored: update 'users' state object
                const userProfile = agentState.get<UserProfile>('userProfile');
                if (userProfile && userProfile.email) {
                    const users = agentState.get<Record<string, any>>('users') || {};
                    users[userProfile.email] = {
                        ...(users[userProfile.email] || {}),
                        email: userProfile.email,
                        totpSecret: secret,
                        hasTotp: true
                    };
                    agentState.set('users', users);
                    console.log(`[Setup] Persisted secret for ${userProfile.email}`);
                }

                // Update local profile instance if needed for remainder of session
                if (userProfile) {
                    userProfile.totpSecret = secret;
                }

                console.log(`[Setup] Secret: ${secret.substring(0, 4)}... (stored for future use)`);

                const code = TOTPService.generate(secret);
                console.log(`[Setup] Generated Code: ${code}`);

                await totpInput.fill(code);

                console.log('[Setup] Submitting TOTP code...');
                await page.click('button[type="submit"], button:has-text("Save"), button:has-text("Submit")');

                console.log('[Setup] Waiting for navigation...');
                // Wait for URL to change indicating successful setup
                await page.waitForURL(/.*/, { waitUntil: 'commit', timeout: 15000 }).catch(() => {
                    console.log('[Setup] No URL change detected, page may have updated in place');
                });

                const finalUrl = page.url();
                console.log(`[Setup] Final URL: ${finalUrl}`);
            }
        }
    });

    // Skip TOTP setup and go directly to Security Key registration
    graph.addTransition({
        from: 'setup-secure',
        to: 'webauthn-register',
        trigger: {
            id: 'skip-totp-register-webauthn',
            description: 'Skip TOTP and register WebAuthn directly',
            condition: async (page: Page) => {
                const isSetupPage = page.url().includes('setup_secure');
                if (!isSetupPage) return false;

                // Check if we can see a Security Key link to skip TOTP
                const hasSecurityKeyLink = await page.locator('a:has-text("Security key")').isVisible().catch(() => false);

                return hasSecurityKeyLink;
            },
            action: async (page: Page) => {
                console.log('[Setup] Skipping TOTP setup, navigating to Security Key registration...');
                await page.click('a:has-text("Security key")');
                await page.waitForLoadState('networkidle');
            }
        }
    });

    // After TOTP setup, handle the updated page (URL stays same but content changes)
    graph.addTransition({
        from: 'setup-secure',
        to: 'dashboard',
        trigger: {
            id: 'setup-totp-complete',
            description: 'TOTP Setup Complete - Navigate Away',
            condition: async (page: Page) => {
                // We're on setup_secure page but TOTP input/secret are gone (setup complete)
                const hasTotpInput = await page.locator('input[name="totp_code"]').isVisible();
                const hasSecretKey = await page.locator('[data-testid="node/text/totp_secret_key/text"]').isVisible();

                console.log(`[Setup] Checking completion: hasTotpInput=${hasTotpInput}, hasSecretKey=${hasSecretKey}`);

                // If both are gone, TOTP might be set up
                if (!hasTotpInput && !hasSecretKey) {
                    // Dump page content for debugging
                    const bodyText = await page.locator('body').innerText().catch(() => 'Unable to get body text');
                    console.log(`[Setup] Page content after TOTP submit:\n${bodyText.substring(0, 500)}`);

                    // Check what's actually visible
                    const hasReturnLink = await page.locator('a[href*="return_to"]').count().catch(() => 0);
                    const hasContinueLink = await page.getByRole('link', { name: /Continue/i }).count().catch(() => 0);
                    const hasCloseMessage = await page.getByText(/close this tab/i).count().catch(() => 0);
                    const hasAnyButton = await page.locator('button').count().catch(() => 0);
                    const hasAnyLink = await page.locator('a').count().catch(() => 0);

                    console.log(`[Setup] Elements found: returnLink=${hasReturnLink}, continueLink=${hasContinueLink}, closeMessage=${hasCloseMessage}, buttons=${hasAnyButton}, links=${hasAnyLink}`);

                    // If there are buttons or links, list them
                    if (hasAnyButton > 0) {
                        const buttons = await page.locator('button').allInnerTexts();
                        console.log(`[Setup] Buttons: ${JSON.stringify(buttons)}`);
                    }
                    if (hasAnyLink > 0) {
                        const links = await page.locator('a').allInnerTexts();
                        console.log(`[Setup] Links: ${JSON.stringify(links.slice(0, 10))}`); // First 10 links
                    }

                    // Return true always to proceed (since TOTP is done)
                    return true;
                }

                return false;
            },
            action: async (page: Page) => {
                console.log('[Setup] TOTP setup complete, looking for navigation...');

                // Try to click return/continue link
                const returnLink = page.locator('a[href*="return_to"]').first();
                const continueLink = page.getByRole('link', { name: /Continue immediately/i });

                if (await returnLink.isVisible().catch(() => false)) {
                    console.log('[Setup] Clicking return link...');
                    await returnLink.click();
                    await page.waitForLoadState('networkidle');
                } else if (await continueLink.isVisible().catch(() => false)) {
                    console.log('[Setup] Clicking continue link...');
                    await continueLink.click();
                    await page.waitForLoadState('networkidle');
                } else {
                    console.log('[Setup] No navigation link found, manually navigating to dashboard...');
                    // Navigate manually to dashboard since setup is complete
                    await page.goto('http://localhost/ui/welcome');
                    await page.waitForLoadState('networkidle');
                }
            }
        }
    });

    // Setup Complete -> Continue or auto-redirect
    graph.addTransition({
        from: 'setup-complete',
        to: 'dashboard',
        trigger: {
            id: 'setup-complete-continue',
            description: 'Continue from setup complete page',
            condition: async (page: Page) => {
                // Check for "Continue immediately" link or wait for auto-redirect
                const continueLink = page.getByRole('link', { name: /Continue immediately/i });
                return await continueLink.isVisible();
            },
            action: async (page: Page) => {
                console.log('[SetupComplete] Clicking continue link...');

                const continueLink = page.getByRole('link', { name: /Continue immediately/i });
                await continueLink.click();
                await page.waitForLoadState('networkidle');
            }
        }
    });

    // Manage Secure -> Continue to next page
    graph.addTransition({
        from: 'manage-secure',
        to: 'dashboard',
        trigger: {
            id: 'manage-secure-continue',
            description: 'Continue from MFA management page',
            condition: async (page: Page) => {
                // Check for "Continue" button or link to return
                return (await page.getByRole('link', { name: 'Continue' }).isVisible()) ||
                       (await page.getByRole('button', { name: 'Continue' }).isVisible()) ||
                       (await page.locator('a[href*="return_to"], a[href*="welcome"]').isVisible());
            },
            action: async (page: Page) => {
                console.log('[ManageSecure] Continuing to next page...');

                // Try to click continue button/link
                const continueLink = page.getByRole('link', { name: 'Continue' });
                const continueButton = page.getByRole('button', { name: 'Continue' });
                const returnLink = page.locator('a[href*="return_to"], a[href*="welcome"]').first();

                if (await continueLink.isVisible()) {
                    await continueLink.click();
                } else if (await continueButton.isVisible()) {
                    await continueButton.click();
                } else if (await returnLink.isVisible()) {
                    await returnLink.click();
                }

                await page.waitForLoadState('networkidle');
            }
        }
    });
}
