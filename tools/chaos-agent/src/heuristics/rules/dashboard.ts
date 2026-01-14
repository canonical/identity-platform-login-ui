import { Graph } from '../../core/graph';
import { Page } from 'playwright';
import { agentState } from '../../core/state';
import { UserProfile } from '../../core/user-profiles';
import chalk from 'chalk';

export function dashboardRules(graph: Graph) {
    // The "dashboard" is actually any manage_* page with SelfServeNavigation
    // Also includes setup_backup_codes (used by manage_* pages)
    // NOTE: reset_password is used by RECOVERY flow, not dashboard!
    graph.addState({
        id: 'dashboard',
        urlPattern: /.*\/ui\/(manage_details|manage_password|manage_backup_codes|manage_secure|manage_connected_accounts|setup_backup_codes).*/,
        name: 'Dashboard (Secure)'
    });

    // 1. HIGH PRIORITY: Logout (Simulated via Cookie Deletion) for WebAuthn Flow
    // Checked FIRST.
    graph.addTransition({
        from: 'dashboard',
        to: 'login',
        trigger: {
            id: 'logout-cookie-clear',
            description: 'Sign out (Simulated via Cookie Deletion)',
            condition: async (page: Page) => {
                const currentProfile = agentState.get<UserProfile>('userProfile');

                // Only WebAuthn flow should force this specific logout path
                if (currentProfile?.name !== 'webauthn-flow') {
                    return false;
                }

                // Check if we have registered keys successfully
                const isRegistered = agentState.get('webauthnRegistered') === true;

                if (isRegistered) {
                     console.log(chalk.yellow(`[Dashboard] WebAuthn Flow: Key registered. Ready to logout.`));
                }

                return isRegistered;
            },
            action: async (page: Page) => {
                console.log(chalk.blue('[Dashboard] Signing out by clearing Kratos session cookies...'));

                const context = page.context();
                const cookies = await context.cookies();
                const kratosCookies = cookies.filter(c => c.name.includes('ory_kratos_session'));

                // Nuclear option: Clear everything
                await context.clearCookies();
                await page.evaluate(() => {
                    try {
                        localStorage.clear();
                        sessionStorage.clear();
                    } catch (e) {}
                });

                console.log(`[Dashboard] Cleared ${kratosCookies.length} Kratos cookies and all storage.`);

                // Force navigation to login (simulating user return)
                console.log('[Dashboard] Navigating to Login page...');
                await page.goto('http://localhost/ui/login');

                // Wait for login page to settle
                try {
                    await page.waitForURL(/.*\/ui\/login.*/, { timeout: 10000 });
                    console.log('[Dashboard] Successfully reached Login URL');
                } catch (e) {
                     console.log('[Dashboard] Warning: Did not detect login URL after forced navigation.');
                }
            },
            weight: 100
        }
    });

    // 2. Navigate to Security Key (WebAuthn Flow Preferred Action)
    graph.addTransition({
        from: 'dashboard',
        to: 'webauthn-register',
        trigger: {
            id: 'goto-security-key',
            description: 'Navigate to Security Key',
            condition: async (page: Page) => {
                const link = page.locator('a[href*="manage_passkey"]');
                const isVisible = await link.isVisible().catch(() => false);
                const currentUrl = page.url();
                return isVisible && !currentUrl.includes('manage_passkey') && !currentUrl.includes('setup_passkey');
            },
            action: async (page: Page) => {
                console.log('[Dashboard] Navigating to Security Key...');
                await page.click('a[href*="manage_passkey"]');
                await page.waitForLoadState('networkidle');
            }
        }
    });

    // 3. Other Dashboard Actions (Disabled for webauthn-flow to prevent distractions)

    // Dashboard -> Navigate to Personal Details
    graph.addTransition({
        from: 'dashboard',
        to: 'dashboard',
        trigger: {
            id: 'goto-personal-details',
            description: 'Navigate to Personal Details',
            condition: async (page: Page) => {
                // SKIP for webauthn-flow
                const currentProfile = agentState.get<UserProfile>('userProfile');
                if (currentProfile?.name === 'webauthn-flow') return false;

                const link = page.locator('a[href*="manage_details"]');
                const isVisible = await link.isVisible().catch(() => false);
                const currentUrl = page.url();
                // Don't navigate if already on this page
                return isVisible && !currentUrl.includes('manage_details');
            },
            action: async (page: Page) => {
                console.log('[Dashboard] Navigating to Personal Details...');
                await page.click('a[href*="manage_details"]');
                await page.waitForLoadState('networkidle');
            }
        }
    });

    // Dashboard -> Navigate to Password Management
    graph.addTransition({
        from: 'dashboard',
        to: 'dashboard',
        trigger: {
            id: 'goto-password',
            description: 'Navigate to Password Management',
            condition: async (page: Page) => {
                 // SKIP for webauthn-flow
                 const currentProfile = agentState.get<UserProfile>('userProfile');
                 if (currentProfile?.name === 'webauthn-flow') return false;

                const link = page.locator('a[href*="manage_password"]');
                const isVisible = await link.isVisible().catch(() => false);
                const currentUrl = page.url();
                // Don't navigate if already on password or reset_password page
                return isVisible && !currentUrl.includes('manage_password') && !currentUrl.includes('reset_password');
            },
            action: async (page: Page) => {
                console.log('[Dashboard] Navigating to Password Management...');
                await page.click('a[href*="manage_password"]');
                await page.waitForLoadState('networkidle');
            }
        }
    });

    // Dashboard -> Navigate to Backup Codes
    graph.addTransition({
        from: 'dashboard',
        to: 'dashboard',
        trigger: {
            id: 'goto-backup-codes',
            description: 'Navigate to Backup Codes',
            condition: async (page: Page) => {
                 // SKIP for webauthn-flow
                 const currentProfile = agentState.get<UserProfile>('userProfile');
                 if (currentProfile?.name === 'webauthn-flow') return false;

                const link = page.locator('a[href*="manage_backup_codes"]');
                const isVisible = await link.isVisible().catch(() => false);
                const currentUrl = page.url();
                return isVisible && !currentUrl.includes('manage_backup_codes') && !currentUrl.includes('backup_codes');
            },
            action: async (page: Page) => {
                console.log('[Dashboard] Navigating to Backup Codes...');
                await page.click('a[href*="manage_backup_codes"]');
                await page.waitForLoadState('networkidle');
            }
        }
    });

    // Dashboard -> Navigate to Authenticator (TOTP)
    graph.addTransition({
        from: 'dashboard',
        to: 'dashboard',
        trigger: {
            id: 'goto-authenticator',
            description: 'Navigate to Authenticator',
            condition: async (page: Page) => {
                 // SKIP for webauthn-flow
                 const currentProfile = agentState.get<UserProfile>('userProfile');
                 if (currentProfile?.name === 'webauthn-flow') return false;

                const link = page.locator('a[href*="manage_secure"]');
                const isVisible = await link.isVisible().catch(() => false);
                const currentUrl = page.url();
                return isVisible && !currentUrl.includes('manage_secure');
            },
            action: async (page: Page) => {
                console.log('[Dashboard] Navigating to Authenticator...');
                await page.click('a[href*="manage_secure"]');
                await page.waitForLoadState('networkidle');
            }
        }
    });

    // Dashboard -> Navigate to Connected Accounts
    graph.addTransition({
        from: 'dashboard',
        to: 'dashboard',
        trigger: {
            id: 'goto-connected-accounts',
            description: 'Navigate to Connected Accounts',
            condition: async (page: Page) => {
                 // SKIP for webauthn-flow
                 const currentProfile = agentState.get<UserProfile>('userProfile');
                 if (currentProfile?.name === 'webauthn-flow') return false;

                const link = page.locator('a[href*="manage_connected_accounts"]');
                const isVisible = await link.isVisible().catch(() => false);
                const currentUrl = page.url();
                return isVisible && !currentUrl.includes('manage_connected_accounts');
            },
            action: async (page: Page) => {
                console.log('[Dashboard] Navigating to Connected Accounts...');
                await page.click('a[href*="manage_connected_accounts"]');
                await page.waitForLoadState('networkidle');
            }
        }
    });
}
