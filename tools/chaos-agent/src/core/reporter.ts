import { Page, BrowserContext } from 'playwright';
import chalk from 'chalk';
import * as fs from 'fs';
import * as path from 'path';

export class Reporter {
  private outputDir: string;

  constructor(outputDir: string = 'test-results') {
    this.outputDir = outputDir;
    if (!fs.existsSync(this.outputDir)) {
      fs.mkdirSync(this.outputDir, { recursive: true });
    }
  }

  logInfo(message: string) {
    console.log(chalk.blue(`[INFO] ${message}`));
  }

  logSuccess(message: string) {
    console.log(chalk.green(`[SUCCESS] ${message}`));
  }

  logStep(message: string) {
    console.log(chalk.gray(`[STEP] ${message}`));
  }

  logError(message: string, error?: unknown) {
    console.error(chalk.red(`[ERROR] ${message}`), error || '');
  }

  async startTracing(context: BrowserContext) {
      await context.tracing.start({ screenshots: true, snapshots: true });
  }

  async stopTracing(context: BrowserContext, name: string) {
      const tracePath = path.join(this.outputDir, `${name}-trace.zip`);
      await context.tracing.stop({ path: tracePath });
      this.logInfo(`Trace saved to: ${tracePath}`);
  }

  async captureFailure(page: Page, testName: string) {
    const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
    const screenshotPath = path.join(this.outputDir, `${testName}-fail-${timestamp}.png`);
    
    try {
      await page.screenshot({ path: screenshotPath, fullPage: true });
      this.logInfo(`Screenshot saved to: ${screenshotPath}`);
    } catch (e) {
      this.logError('Failed to capture screenshot', e);
    }
  }
}
