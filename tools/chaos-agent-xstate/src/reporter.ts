import fs from 'fs';
import path from 'path';
import { Page } from 'playwright';
import chalk from 'chalk';

interface StepReport {
    step: string;
    timestamp: string;
    status: 'success' | 'failure' | 'pending';
    screenshotPath?: string;
    error?: string;
    logs: string[];
}

interface TestReport {
    id: string;
    startTime: string;
    duration: number;
    status: 'passed' | 'failed' | 'running';
    videoPath?: string;
    steps: StepReport[];
}

export class Reporter {
    private report: TestReport;
    public reportDir: string; // Made public for use in index.ts
    private page: Page | undefined;
    private currentStepIndex: number = -1;

    constructor() {
        const timestamp = new Date().toISOString().replace(/:/g, '-');
        this.reportDir = path.join(process.cwd(), 'reports', timestamp);

        if (!fs.existsSync(this.reportDir)) {
            fs.mkdirSync(this.reportDir, { recursive: true });
        }

        this.report = {
            id: `test-${timestamp}`,
            startTime: new Date().toISOString(),
            duration: 0,
            status: 'running',
            steps: []
        };
    }

    setPage(page: Page) {
        this.page = page;
    }

    setVideoPath(videoFileName: string) {
        this.report.videoPath = videoFileName;
    }

    logStep(step: string) {
        const timestamp = new Date().toISOString();
        console.log(chalk.cyan(`[Reporter] Step: ${step}`));
        this.report.steps.push({
            step,
            timestamp,
            status: 'success', // optimistically set, updated on failure
            logs: []
        });
        this.currentStepIndex = this.report.steps.length - 1;
    }

    logInfo(message: string) {
        console.log(message); // Still print to console
        if (this.currentStepIndex >= 0) {
            // Strip ANSI codes for the report
            const cleanMessage = message.replace(/[\u001b\u009b][[()#;?]*(?:[0-9]{1,4}(?:;[0-9]{0,4})*)?[0-9A-ORZcf-nqry=><]/g, '');
            this.report.steps[this.currentStepIndex].logs.push(cleanMessage);
        }
    }

    async logFailure(error: string) {
        this.report.status = 'failed';
        if (this.report.steps.length > 0) {
            const lastStep = this.report.steps[this.report.steps.length - 1];
            lastStep.status = 'failure';
            lastStep.error = error;

            if (this.page) {
                const screenshotName = `failure-${Date.now()}.png`;
                const screenshotPath = path.join(this.reportDir, screenshotName);
                try {
                    await this.page.screenshot({ path: screenshotPath, fullPage: true });
                    lastStep.screenshotPath = screenshotName;
                    console.log(chalk.red(`[Reporter] Screenshot saved to ${screenshotPath}`));
                } catch (e) {
                    console.log(chalk.red('[Reporter] Failed to take screenshot: ' + e));
                }
            }
        }
    }

    async generateReport() {
        const endTime = new Date();
        const startTime = new Date(this.report.startTime);
        this.report.duration = (endTime.getTime() - startTime.getTime()) / 1000;

        if (this.report.status === 'running') {
            this.report.status = 'passed';
        }

        const reportPath = path.join(this.reportDir, 'report.json');
        fs.writeFileSync(reportPath, JSON.stringify(this.report, null, 2));
        console.log(chalk.green(`[Reporter] Report generated at ${reportPath}`));
    }
}
