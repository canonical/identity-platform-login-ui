import { Graph } from '../../core/graph';
import { Page } from 'playwright';
import { EmailService } from '../../core/services/email';

const emailService = new EmailService();
let initialEmailCount: number | undefined;

export function recoveryRules(graph: Graph) {
    graph.addState({
        id: 'recovery-email',
        urlPattern: /.*\/ui\/reset_email.*/,
        name: 'Recovery - Enter Email',
        contentCheck: async (page: Page) => {
            // Check if we're on the email entry step (not code entry) - use count to avoid timeouts
            const emailCount = await page.locator('input[type="email"]').count();
            const codeCount = await page.locator('input[name="code"]').count();
            return emailCount > 0 && codeCount === 0;
        }
    });

    graph.addState({
        id: 'recovery-code',
        urlPattern: /.*\/ui\/reset_email.*/,
        name: 'Recovery - Enter Code',
        contentCheck: async (page: Page) => {
            // Check if we're on the code entry step - use count to avoid visibility timeouts
            return (await page.locator('input[name="code"]').count()) > 0;
        }
    });

    graph.addState({
        id: 'recovery-password',
        urlPattern: /.*\/ui\/reset_password.*/,
        name: 'Recovery - Set New Password'
    });

    // Transition 1: Login -> Recovery Email Entry
    graph.addTransition({
        from: 'login',
        to: 'recovery-email',
        trigger: {
            id: 'goto-recovery',
            description: 'Click Forgot Password',
            condition: async (page: Page) => {
                const forgotLink = page.locator('a:has-text("Forgot password"), a:has-text("reset password")');
                return await forgotLink.isVisible();
            },
            action: async (page: Page) => {
                console.log('[Recovery] Navigating to password recovery...');
                await page.click('a:has-text("Forgot password"), a:has-text("reset password")');
                await page.waitForLoadState('networkidle');
                console.log('[Recovery] Arrived at email entry page');
            }
        }
    });

    // Transition 2: Recovery Email Entry -> Recovery Code Entry
    graph.addTransition({
        from: 'recovery-email',
        to: 'recovery-code',
        trigger: {
            id: 'recovery-submit-email',
            description: 'Submit Recovery Email',
            condition: async (page: Page) => {
                // Simple check: if title contains "Enter an email"
                const title = await page.title();
                return title.includes('Enter an email');
            },
            action: async (page: Page) => {
                const email = 'test@example.com';
                console.log(`[Recovery] Submitting recovery email: ${email}`);
                
                // Count emails BEFORE submitting to detect new email arrival
                initialEmailCount = await emailService.getEmailCount(email);
                console.log(`[Recovery] Current email count: ${initialEmailCount}`);
                
                // Fill email input (always type="email" in this UI)
                await page.locator('input[type="email"]').fill(email);
                // Use text-based selector since button doesn't have type="submit"
                await page.getByRole('button', { name: 'Reset password' }).click();
                
                // Wait for code input to appear (indicates form has updated)
                await page.locator('input[name="code"]').waitFor({ state: 'visible', timeout: 10000 });
                console.log('[Recovery] Email submitted, code entry form loaded');
            }
        }
    });

    // Transition 3: Recovery Code Entry -> New Password Entry
    graph.addTransition({
        from: 'recovery-code',
        to: 'recovery-password',
        trigger: {
            id: 'recovery-verify-code',
            description: 'Fetch and Enter Recovery Code',
            condition: async (page: Page) => {
                const codeInput = page.locator('input[name="code"]');
                return await codeInput.isVisible();
            },
            action: async (page: Page) => {
                console.log('[Recovery] Fetching recovery code from Mailslurper...');
                
                // Wait for a NEW email (count should increase from initialEmailCount)
                const code = await emailService.getLatestCode('test@example.com', 10, 1000, initialEmailCount);
                
                if (!code) {
                    throw new Error('[Recovery] Could not fetch recovery code from Mailslurper');
                }
                
                console.log(`[Recovery] Found code: ${code}`);
                await page.fill('input[name="code"]', code);
                // Use text-based selector for submit button
                await page.getByRole('button', { name: /submit/i }).click();
                
                // Wait for navigation to password reset page
                await page.waitForURL(/.*\/ui\/reset_password.*/, { timeout: 10000 });
                console.log('[Recovery] Code verified, navigated to password reset page');
            }
        }
    });

    // Transition 4: New Password Entry -> Login (or Dashboard if auto-login)
    graph.addTransition({
        from: 'recovery-password',
        to: 'login',
        trigger: {
            id: 'recovery-set-password',
            description: 'Set New Password',
            condition: async (page: Page) => {
                const passwordInput = page.locator('input[name="password"], input[type="password"]').first();
                const submitBtn = page.locator('button[type="submit"]');
                return (await passwordInput.isVisible()) && (await submitBtn.isVisible());
            },
            action: async (page: Page) => {
                const newPassword = 'NewPassword123!';
                console.log('[Recovery] Setting new password...');
                
                // Fill password field(s) - there may be two (password + confirm)
                const passwordInputs = page.locator('input[name="password"], input[type="password"]');
                const count = await passwordInputs.count();
                
                for (let i = 0; i < count; i++) {
                    await passwordInputs.nth(i).fill(newPassword);
                }
                
                // Use first button (should be the submit button)
                await page.locator('button').first().click();
                
                // Wait for navigation to login page (recovery complete)
                await page.waitForURL(/.*\/ui\/login.*/, { timeout: 10000 });
                console.log('[Recovery] ✅ Password reset complete, redirected to login');
            }
        }
    });
}
