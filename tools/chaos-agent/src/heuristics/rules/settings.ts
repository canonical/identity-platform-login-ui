import { Graph } from '../../core/graph';
import { Page } from 'playwright';

export function settingsRules(graph: Graph) {
    // State: Settings Page
    graph.addState({
        id: 'settings',
        urlPattern: /.*\/ui\/settings.*/,
        name: 'User Settings'
    });

    // 1. Remove WebAuthn
    graph.addTransition({
        from: 'settings',
        to: 'settings', // Stays on page
        trigger: {
            id: 'remove-webauthn',
            description: 'Remove a Security Key',
            condition: async (page: Page) => {
                // Look for trash icon or delete button next to a key
                return await page.locator('button[aria-label="Remove security key"]').count() > 0;
            },
            action: async (page: Page) => {
                await page.click('button[aria-label="Remove security key"] >> nth=0');
                await page.click('button:has-text("Confirm")'); // Modal confirmation
                await page.waitForLoadState('networkidle');
                console.log('[Settings] Removed a Security Key');
            }
        }
    });

    // 2. Remove TOTP
    graph.addTransition({
        from: 'settings',
        to: 'settings',
        trigger: {
            id: 'remove-totp',
            description: 'Remove Authenticator App',
            condition: async (page: Page) => {
                return await page.locator('button[aria-label="Remove authenticator app"]').isVisible();
            },
            action: async (page: Page) => {
                await page.click('button[aria-label="Remove authenticator app"]');
                await page.click('button:has-text("Confirm")');
                await page.waitForLoadState('networkidle');
                console.log('[Settings] Removed TOTP App');
            }
        }
    });
}
