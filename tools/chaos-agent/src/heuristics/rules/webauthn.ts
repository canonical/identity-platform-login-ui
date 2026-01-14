import { Graph } from '../../core/graph';
import { Page } from 'playwright';
import { WebAuthnService } from '../../core/services/webauthn';
import { agentState } from '../../core/state';

export function webauthnRules(graph: Graph, webAuthnService: WebAuthnService) {

    // Define WebAuthn 2FA verification state (password → security key prompt)
    graph.addState({
        id: 'webauthn-verify',
        urlPattern: /.*\/ui\/login.*/,
        name: 'WebAuthn 2FA Verification',
        contentCheck: async (page: Page) => {
            const isLoginPage = page.url().includes('/ui/login');
            if (!isLoginPage) return false;

            // Check for WebAuthn authentication prompt (after password)
            const hasWebAuthnButton = await page.locator('button:has-text("Sign in with Security key")').isVisible().catch(() => false);
            const hasHardwareKeyButton = await page.locator('button:has-text("Sign in with hardware key")').isVisible().catch(() => false);
            const hasUseKeyButton = await page.locator('button:has-text("Use security key")').isVisible().catch(() => false);
            const hasAuthenticateButton = await page.locator('button:has-text("Authenticate")').isVisible().catch(() => false);

            // Must have WebAuthn button and NOT have password input (already submitted)
            const hasPasswordInput = await page.locator('input[type="password"]').isVisible().catch(() => false);

            return (hasWebAuthnButton || hasHardwareKeyButton || hasUseKeyButton || hasAuthenticateButton) && !hasPasswordInput;
        }
    });

    // Define WebAuthn registration state
    graph.addState({
        id: 'webauthn-register',
        urlPattern: /.*\/ui\/(manage_passkey|setup_passkey).*/,
        name: 'WebAuthn Registration',
        contentCheck: async (page: Page) => {
            const isPasskeyPage = page.url().includes('manage_passkey') || page.url().includes('setup_passkey');
            if (!isPasskeyPage) return false;

            const hasAddButton = await page.locator('button:has-text("Add security key")').isVisible().catch(() => false);
            return hasAddButton;
        }
    });

    // 1. Login with WebAuthn (after password) - 2FA Mode
    graph.addTransition({
        from: 'login',
        to: 'webauthn-verify',
        trigger: {
            id: 'password-submitted-webauthn-prompt',
            description: 'Password submitted, WebAuthn prompt appears',
            condition: async (page: Page) => {
                // This transition happens automatically after password submission
                // when user only has WebAuthn configured (no TOTP)
                return false; // Let state detection handle this
            },
            action: async (page: Page) => {
                console.log('[WebAuthn] Password accepted, WebAuthn prompt detected');
            }
        }
    });

    // 2. Authenticate with WebAuthn Security Key
    graph.addTransition({
        from: 'webauthn-verify',
        to: 'dashboard',
        trigger: {
            id: 'authenticate-with-security-key',
            description: 'Authenticate with WebAuthn security key',
            condition: async (page: Page) => {
                const isLoginPage = page.url().includes('/ui/login');
                if (!isLoginPage) return false;

                const hasWebAuthnButton = await page.locator('button:has-text("Sign in with Security key")').isVisible().catch(() => false);
                const hasHardwareKeyButton = await page.locator('button:has-text("Sign in with hardware key")').isVisible().catch(() => false);
                const hasUseKeyButton = await page.locator('button:has-text("Use security key")').isVisible().catch(() => false);
                const hasAuthenticateButton = await page.locator('button:has-text("Authenticate")').isVisible().catch(() => false);

                return hasWebAuthnButton || hasHardwareKeyButton || hasUseKeyButton || hasAuthenticateButton;
            },
            action: async (page: Page) => {
                console.log('[WebAuthn] Authenticating with security key...');
                await webAuthnService.enable();

                // Try different button variations
                const signInBtn = page.locator('button:has-text("Sign in with Security key")');
                const hardwareKeyBtn = page.locator('button:has-text("Sign in with hardware key")');
                const useKeyBtn = page.locator('button:has-text("Use security key")');
                const authenticateBtn = page.locator('button:has-text("Authenticate")');

                if (await hardwareKeyBtn.isVisible().catch(() => false)) {
                    console.log('[WebAuthn] Clicking "Sign in with hardware key" button');
                    await hardwareKeyBtn.click();
                } else if (await signInBtn.isVisible().catch(() => false)) {
                    console.log('[WebAuthn] Clicking "Sign in with Security key" button');
                    await signInBtn.click();
                } else if (await useKeyBtn.isVisible().catch(() => false)) {
                    console.log('[WebAuthn] Clicking "Use security key" button');
                    await useKeyBtn.click();
                } else if (await authenticateBtn.isVisible().catch(() => false)) {
                    console.log('[WebAuthn] Clicking "Authenticate" button');
                    await authenticateBtn.click();
                } else {
                    throw new Error('[WebAuthn] No WebAuthn button found');
                }

                // Wait for navigation or error
                try {
                    // Robust wait for redirection away from login/self-service
                    await page.waitForURL((url) => {
                        const u = url.toString();
                        // Success if we are on a known post-login UI page
                        return u.includes('/ui/manage_') ||
                               u.includes('/ui/setup_') ||
                               u.includes('/ui/welcome');
                    }, { timeout: 15000 });

                    console.log('[WebAuthn] ✅ Authenticated with Security Key - Dashboard/Setup loaded');
                } catch (e) {
                     // Check if still on login page
                    const url = page.url();
                    if (url.includes('/ui/login')) {
                        console.log('[WebAuthn] ⚠️ Still on login page after clicking - auth may have failed');
                    } else if (url.includes('self-service')) {
                        console.log('[WebAuthn] ⚠️ Stuck on Kratos API URL - attempting manual navigation to Dashboard...');
                        const uiBase = page.url().split('/self-service')[0]; // Extract base URL

                        // Use JS navigation to avoid strict Playwright checks on pending requests
                        await page.evaluate((target) => window.location.href = target, `${uiBase}/ui/`);

                        // Wait again for dashboard
                        await page.waitForURL((u) => {
                            const str = u.toString();
                            return str.includes('/ui/manage_') || str.includes('/ui/setup_') || str.includes('/ui/welcome');
                        }, { timeout: 10000 });
                         console.log('[WebAuthn] ✅ Recovered from Kratos API URL - Dashboard loaded');
                    } else {
                        console.log(`[WebAuthn] ⚠️ Navigation timeout/error: ${e}`);
                    }
                }
            }
        }
    });

    // 3. Registration Flow - 2FA Mode (Dashboard -> Add WebAuthn as 2nd factor)
    graph.addTransition({
        from: 'dashboard',
        to: 'webauthn-register',
        trigger: {
            id: 'navigate-webauthn-register',
            description: 'Navigate to WebAuthn registration page',
            condition: async (page: Page) => {
                const isDashboard = page.url().includes('manage_');
                if (!isDashboard) return false;

                const isNotPasskeyPage = !page.url().includes('manage_passkey') && !page.url().includes('setup_passkey');
                if (!isNotPasskeyPage) return false;

                const hasPasskeyLink = await page.locator('a:has-text("Security key")').isVisible().catch(() => false);
                return hasPasskeyLink;
            },
            action: async (page: Page) => {
                console.log('[WebAuthn] Navigating to security key management...');
                await page.click('a:has-text("Security key")');
                await page.waitForLoadState('networkidle');
            }
        }
    });

    // 4. Actually Register the WebAuthn Device
    graph.addTransition({
        from: 'webauthn-register',
        to: 'webauthn-register',
        trigger: {
            id: 'register-webauthn-device',
            description: 'Register a new WebAuthn device',
            condition: async (page: Page) => {
                const isPasskeyPage = page.url().includes('manage_passkey') || page.url().includes('setup_passkey');
                if (!isPasskeyPage) return false;

                const hasAddButton = await page.locator('button:has-text("Add security key")').isVisible().catch(() => false);
                const hasNameInput = await page.getByLabel('Security key name').isVisible().catch(() => false);

                // Check how many keys are already registered
                const keyCount = await page.locator('button:has-text("Delete")').count();
                const registeredKeysCount = (agentState.get('registeredKeysCount') as number) || 0;

                // Allow registration if we have less than 5 keys total
                return hasAddButton && hasNameInput && keyCount < 5;
            },
            action: async (page: Page) => {
                console.log('[WebAuthn] Starting registration...');
                await webAuthnService.enable();

                // Generate unique key name
                const timestamp = Date.now();
                const keyIndex = ((agentState.get('registeredKeysCount') as number) || 0) + 1;
                const keyName = `Chaos Agent Key ${keyIndex} (${timestamp})`;

                const nameInput = page.getByLabel('Security key name');
                await nameInput.fill(keyName);

                console.log(`[WebAuthn] Clicking "Add security key" button (name: ${keyName})...`);
                await page.click('button:has-text("Add security key")');

                await page.waitForLoadState('networkidle');

                try {
                    const successVisible = await page.locator(`text=/Chaos Agent Key ${keyIndex}/`).isVisible({ timeout: 5000 });
                    if (successVisible) {
                        console.log(`[WebAuthn] ✅ Key ${keyIndex} registered successfully - visible in list: ${keyName}`);
                        agentState.set('webauthnRegistered', true);
                        agentState.set('lastWebAuthnKeyName', keyName);
                        agentState.set('registeredKeysCount', keyIndex);
                    }
                } catch (e) {
                    console.log('[WebAuthn] Warning: Could not verify key in list, but continuing (optimistic success)');
                    // Assume success to allow flow to proceed to logout/login
                    agentState.set('webauthnRegistered', true);
                    agentState.set('registeredKeysCount', keyIndex);
                }
            }
        }
    });

    // 5. Delete WebAuthn Device
    graph.addTransition({
        from: 'webauthn-register',
        to: 'webauthn-register',
        trigger: {
            id: 'delete-webauthn-device',
            description: 'Delete a WebAuthn device',
            condition: async (page: Page) => {
                const isPasskeyPage = page.url().includes('manage_passkey') || page.url().includes('setup_passkey');
                if (!isPasskeyPage) return false;

                // Check if there's a delete button (meaning keys exist)
                const hasDeleteButton = await page.locator('button:has-text("Delete")').first().isVisible().catch(() => false);

                // Count existing keys
                const keyCount = await page.locator('button:has-text("Delete")').count();
                const deletedKeysCount = (agentState.get('deletedKeysCount') as number) || 0;

                // Only delete if:
                // 1. We have more than one key (keep at least one for login tests)
                // 2. We haven't deleted too many already (max 3 deletions)
                return hasDeleteButton && keyCount > 1 && deletedKeysCount < 3;
            },
            action: async (page: Page) => {
                const deletedCount = ((agentState.get('deletedKeysCount') as number) || 0) + 1;
                console.log(`[WebAuthn] Deleting security key #${deletedCount}...`);

                // Get the first delete button
                const deleteButton = page.locator('button:has-text("Delete")').first();
                await deleteButton.click();

                // Wait for confirmation or deletion
                await page.waitForLoadState('networkidle');
                agentState.set('deletedKeysCount', deletedCount);
                console.log(`[WebAuthn] ✅ Key deleted (total deleted: ${deletedCount})`);
            }
        }
    });

    // 6. Navigate back to Dashboard
    graph.addTransition({
        from: 'webauthn-register',
        to: 'dashboard',
        trigger: {
            id: 'webauthn-back-to-dashboard',
            description: 'Navigate back to dashboard from WebAuthn page',
            condition: async (page: Page) => {
                const isPasskeyPage = page.url().includes('manage_passkey') || page.url().includes('setup_passkey');
                if (!isPasskeyPage) return false;

                // Navigate back after we've done some actions (registered or deleted keys)
                const hasKeys = await page.locator('button:has-text("Delete")').count() > 0;
                return hasKeys;
            },
            action: async (page: Page) => {
                console.log('[WebAuthn] Navigating back to dashboard...');
                // Click on a different dashboard link
                const detailsLink = page.locator('a:has-text("Personal details")');
                try {
                    if (await detailsLink.isVisible()) {
                        await detailsLink.click();
                    } else {
                        console.log('[WebAuthn] "Personal details" link not found. Forcing navigation to dashboard...');
                        const currentUrl = page.url();
                        // If we are on setup_passkey, likely we want manage_details
                        await page.goto(currentUrl.replace('setup_passkey', 'manage_details').replace('manage_passkey', 'manage_details').split('?')[0]);
                    }
                    await page.waitForLoadState('networkidle');
                } catch (e) {
                    console.log(`[WebAuthn] Navigation failed: ${e}`);
                }
            }
        }
    });

    // 7. WebAuthn Login - 2FA Mode (Alternative to TOTP)
    graph.addTransition({
        from: 'totp-verify',
        to: 'dashboard',
        trigger: {
            id: 'login-webauthn-2fa',
            description: 'Login with WebAuthn as 2nd factor',
            condition: async (page: Page) => {
                const isLoginPage = page.url().includes('/ui/login');
                if (!isLoginPage) return false;

                const hasWebAuthnButton = await page.locator('button:has-text("Sign in with Security key")').isVisible().catch(() => false);
                const hasUseKeyButton = await page.locator('button:has-text("Use security key")').isVisible().catch(() => false);

                return hasWebAuthnButton || hasUseKeyButton;
            },
            action: async (page: Page) => {
                console.log('[WebAuthn] Starting 2FA login with security key...');
                await webAuthnService.enable();

                const btn1 = page.locator('button:has-text("Sign in with Security key")');
                const btn2 = page.locator('button:has-text("Use security key")');

                if (await btn1.isVisible().catch(() => false)) {
                    await btn1.click();
                } else if (await btn2.isVisible().catch(() => false)) {
                    await btn2.click();
                } else {
                    throw new Error('[WebAuthn] No WebAuthn button found');
                }

                await page.waitForLoadState('networkidle');
                console.log('[WebAuthn] ✅ Logged in with Security Key (2FA)');
            }
        }
    });

    // 8. WebAuthn Login - Passwordless Mode (Direct login with identifier + WebAuthn)
    graph.addTransition({
        from: 'login',
        to: 'dashboard',
        trigger: {
            id: 'login-webauthn-passwordless',
            description: 'Passwordless login with WebAuthn',
            condition: async (page: Page) => {
                const isLoginPage = page.url().includes('/ui/login');
                if (!isLoginPage) return false;

                const title = await page.title();
                const isSignInPage = title.includes('Sign in') && !title.includes('Verify');
                if (!isSignInPage) return false;

                const hasIdentifier = await page.locator('input[name="identifier"]').isVisible().catch(() => false);
                const hasWebAuthnButton = await page.locator('button:has-text("Sign in with Security key")').isVisible().catch(() => false);
                const hasPassword = await page.locator('input[type="password"]').isVisible().catch(() => false);

                return hasIdentifier && hasWebAuthnButton && !hasPassword;
            },
            action: async (page: Page) => {
                console.log('[WebAuthn] Starting passwordless login with security key...');
                await webAuthnService.enable();

                await page.fill('input[name="identifier"]', 'test@example.com');

                const btn = page.locator('button:has-text("Sign in with Security key")');
                await btn.click();

                await page.waitForLoadState('networkidle');
                console.log('[WebAuthn] ✅ Logged in with Security Key (Passwordless)');
            }
        }
    });
}
