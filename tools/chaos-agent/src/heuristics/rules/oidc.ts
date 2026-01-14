import { Graph } from '../../core/graph';
import { Page } from 'playwright';

export function oidcRules(graph: Graph) {
    // State: OIDC Test Client (Start Page)
    graph.addState({
        id: 'oidc-client',
        urlPattern: /.*:4446\/?$/, // Matches localhost:4446 or 127.0.0.1:4446 root
        name: 'OIDC Test Client'
    });

    // State: OIDC Callback (Success Page)
    graph.addState({
        id: 'oidc-callback',
        urlPattern: /.*:4446\/callback.*/,
        name: 'OIDC Callback Success'
    });

    // 1. Start Login Flow from Client
    graph.addTransition({
        from: 'oidc-client',
        to: 'login', // Transitions to Identity Platform
        trigger: {
            id: 'oidc-start-login',
            description: 'Start OIDC Authorization Code Flow',
            condition: async (page: Page) => {
                // Look for "Authorize application" link on test client
                return await page.locator('a:has-text("Authorize application"), a:has-text("Login"), button:has-text("Login"), button:has-text("Authorize")').isVisible();
            },
            action: async (page: Page) => {
                console.log('[OIDC] Starting Authorization Flow...');
                await page.click('a:has-text("Authorize application"), a:has-text("Login"), button:has-text("Login"), button:has-text("Authorize")');
                await page.waitForLoadState('networkidle');
            }
        }
    });

    // 2. Restart from Callback (Loop)
    graph.addTransition({
        from: 'oidc-callback',
        to: 'oidc-client',
        trigger: {
            id: 'oidc-restart',
            description: 'Restart OIDC Flow',
            condition: async (page: Page) => {
                // Usually a "Home" or "Logout" button
                return await page.locator('a:has-text("Home"), button:has-text("Logout"), a[href="/"]').isVisible();
            },
            action: async (page: Page) => {
                console.log('[OIDC] Flow complete. Restarting...');
                await page.click('a:has-text("Home"), button:has-text("Logout"), a[href="/"]');
                await page.waitForLoadState('networkidle');
            }
        }
    });
}
