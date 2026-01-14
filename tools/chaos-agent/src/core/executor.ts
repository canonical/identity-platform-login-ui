import { Page } from 'playwright';

export class ActionExecutor {
    constructor(private page: Page) {}

    async click(selector: string) {
        console.log(`[Executor] Clicking ${selector}`);
        await this.page.click(selector);
        await this.page.waitForLoadState('networkidle').catch(() => {});
    }

    async fill(selector: string, value: string) {
        console.log(`[Executor] Filling ${selector} with "${value}"`);
        await this.page.fill(selector, value);
    }

    async navigate(url: string) {
        console.log(`[Executor] Navigating to ${url}`);
        await this.page.goto(url);
    }
}
