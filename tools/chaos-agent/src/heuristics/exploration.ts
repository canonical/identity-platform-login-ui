import { Page } from 'playwright';
import { DOMScanner } from '../core/scanner';
import { ActionExecutor } from '../core/executor';
import chalk from 'chalk';

export class ExplorationEngine {
    constructor(
        private page: Page,
        private scanner: DOMScanner,
        private executor: ActionExecutor
    ) {}

    async run(durationSeconds: number = 60) {
        console.log(chalk.magenta('[Exploration] Starting random monkey testing...'));
        const endTime = Date.now() + durationSeconds * 1000;

        while (Date.now() < endTime) {
            // 1. Scan page for interactives
            const result = await this.scanner.scan();
            
            if (result.interactives.length === 0 && result.forms.length === 0) {
                console.log(chalk.yellow('[Exploration] No interactive elements found. Waiting...'));
                await this.page.waitForTimeout(1000);
                continue;
            }

            // 2. Roll dice for action
            // 70% Click existing element
            // 20% Form fill (if forms exist)
            // 10% Chaos (Back/Forward/Refresh)
            const roll = Math.random();

            if (roll < 0.7 && result.interactives.length > 0) {
                // Pick random interactive
                const el = result.interactives[Math.floor(Math.random() * result.interactives.length)];
                console.log(chalk.magenta(`[Exploration] Clicking: ${el.selector} (${el.text})`));
                try {
                    await this.executor.click(el.selector);
                } catch (e) {
                    console.log(chalk.red(`[Exploration] Click failed: ${e}`));
                }
            } else if (roll < 0.9 && result.forms.length > 0) {
                // Fill random form input
                const form = result.forms[Math.floor(Math.random() * result.forms.length)];
                if (form.inputs.length > 0) {
                    const inputName = form.inputs[Math.floor(Math.random() * form.inputs.length)];
                    const value = Math.random().toString(36).substring(7); // Random string
                    const selector = `input[name="${inputName}"]`; // Simplified
                    console.log(chalk.magenta(`[Exploration] Filling ${selector} with ${value}`));
                    try {
                        await this.executor.fill(selector, value);
                    } catch (e) {
                         console.log(chalk.red(`[Exploration] Fill failed: ${e}`));
                    }
                }
            } else {
                // Chaos Navigation
                console.log(chalk.magenta('[Exploration] Chaos Navigation (Back/Refresh)'));
                if (Math.random() > 0.5) {
                    await this.page.goBack().catch(() => {});
                } else {
                    await this.page.reload().catch(() => {});
                }
            }

            await this.page.waitForTimeout(500); // Brief pause
        }
        console.log(chalk.magenta('[Exploration] Finished.'));
    }
}
