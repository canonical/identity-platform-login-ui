import { Page } from 'playwright';
import { ScanResult } from './types';

export class DOMScanner {
    constructor(private page: Page) {}

    async scan(): Promise<ScanResult> {
        // Find buttons, links, inputs
        const interactives = await this.page.evaluate(() => {
            const elements = Array.from(document.querySelectorAll('button, a, input[type="submit"], [role="button"]'));
            return elements.map(el => {
                let text = el.textContent?.trim() || '';
                if (el instanceof HTMLInputElement) {
                    text = el.value || el.placeholder || '';
                }
                
                // Generate a simple CSS selector (best effort)
                let selector = el.tagName.toLowerCase();
                if (el.id) selector += `#${el.id}`;
                else if (el.className) selector += `.${el.className.split(' ').join('.')}`;
                
                return {
                    role: el.tagName.toLowerCase(),
                    selector,
                    text: text.substring(0, 50)
                };
            }).filter(item => item.selector !== ''); // Filter out empty
        });

        // Find forms
        const forms = await this.page.evaluate(() => {
            return Array.from(document.querySelectorAll('form')).map(form => {
                const inputs = Array.from(form.querySelectorAll('input, select, textarea'))
                    .map(i => (i as HTMLElement).getAttribute('name') || '')
                    .filter(n => n !== '');
                
                let selector = 'form';
                if (form.id) selector += `#${form.id}`;
                
                return { selector, inputs };
            });
        });

        return { interactives, forms };
    }
}
