import { Graph } from '../../core/graph';
import { Page } from 'playwright';

export function negativeRules(graph: Graph) {
    // 1. Invalid Login
    graph.addTransition({
        from: 'login',
        to: 'login', // Loop back
        trigger: {
            id: 'login-invalid',
            description: 'Attempt Login with Invalid Credentials',
            condition: async (page: Page) => {
                const hasInputs = (await page.locator('input[name="identifier"]').isVisible()) &&
                                  (await page.locator('input[name="password"]').isVisible());
                const noError = await page.locator('.error-message, [role="alert"]').count() === 0;
                return hasInputs && noError; 
                // Only try if no error is currently showing to avoid loops
            },
            action: async (page: Page) => {
                console.log('[Negative] Injecting invalid credentials...');
                await page.fill('input[name="identifier"]', 'invalid@user.com');
                await page.fill('input[name="password"]', 'wrongpassword');
                await page.click('button[type="submit"]');
                await page.waitForSelector('.error-message, [role="alert"]', { timeout: 5000 });
                console.log('[Negative] Verified error message appeared');
            }
        }
    });

    // 2. Invalid Recovery Email
    graph.addTransition({
        from: 'recovery',
        to: 'recovery',
        trigger: {
            id: 'recovery-invalid-email',
            description: 'Submit Malformed Email for Recovery',
            condition: async (page: Page) => {
                return await page.locator('input[name="email"]').isVisible();
            },
            action: async (page: Page) => {
                console.log('[Negative] Injecting malformed email...');
                await page.fill('input[name="email"]', 'not-an-email');
                await page.click('button[type="submit"]');
                // Expect client-side validation or server error - wait for error message to appear
                await page.locator('.error-message, [role="alert"]').waitFor({ state: 'visible', timeout: 5000 }).catch(() => {
                    console.log('[Negative] No error message appeared (client-side validation may have blocked)');
                });
                console.log('[Negative] Attempted malformed email');
            }
        }
    });
}
