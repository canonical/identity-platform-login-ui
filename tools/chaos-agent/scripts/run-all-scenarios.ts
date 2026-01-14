#!/usr/bin/env node
import { spawn } from 'child_process';
import * as fs from 'fs';
import * as path from 'path';
import chalk from 'chalk';

interface ScenarioResult {
    name: string;
    command: string;
    exitCode: number;
    duration: number;
    output: string;
}

const scenarios = [
    {
        name: 'OIDC Flow',
        description: 'Validates standard OIDC login flow with TOTP',
        args: ['--mode=validation', '--url=http://127.0.0.1:4446/', '--profile=totp-user', '--duration=120']
    },
    {
        name: 'Standalone Login',
        description: 'Validates direct login to Identity Platform with TOTP',
        args: ['--mode=validation', '--url=http://localhost/ui/login', '--profile=totp-user', '--duration=60']
    },
    {
        name: 'Password Recovery',
        description: 'Validates forgot password flow via email',
        args: ['--mode=validation', '--url=http://localhost/ui/reset_email', '--duration=60']
    },
    {
        name: 'WebAuthn Lifecycle',
        description: 'Validates WebAuthn registration, logout, and 2FA login',
        args: ['--mode=validation', '--url=http://localhost/ui/login', '--profile=webauthn-flow', '--duration=90']
    }
];

async function runScenario(scenario: { name: string, args: string[] }): Promise<ScenarioResult> {
    console.log(chalk.bold.blue(`\n[Runner] Starting Scenario: ${scenario.name}`));
    console.log(chalk.gray(`Command: npx ts-node src/cli/index.ts ${scenario.args.join(' ')}`));

    const startTime = Date.now();

    return new Promise((resolve) => {
        // Run in shell mode to ensure npx executable is found and env vars are handled
        const child = spawn('npx', ['ts-node', 'src/cli/index.ts', ...scenario.args], {
            cwd: process.cwd(),
            env: { ...process.env, FORCE_COLOR: '1' }, // Ensure chalk colors are preserved in piped output
            shell: true
        });

        let output = '';

        child.stdout.on('data', (data) => {
            const str = data.toString();
            output += str;
            process.stdout.write(str); // Stream live output to console
        });

        child.stderr.on('data', (data) => {
            const str = data.toString();
            output += str;
            process.stderr.write(str);
        });

        child.on('close', (code) => {
            const duration = (Date.now() - startTime) / 1000;
            const success = code === 0;
            console.log(chalk.bold(success ? chalk.green(`[Runner] ${scenario.name} PASSED (${duration.toFixed(1)}s)`) : chalk.red(`[Runner] ${scenario.name} FAILED (${duration.toFixed(1)}s)`)));
            resolve({
                name: scenario.name,
                command: `npx ts-node src/cli/index.ts ${scenario.args.join(' ')}`,
                exitCode: code || 0,
                duration,
                output
            });
        });
    });
}

function generateReport(results: ScenarioResult[]) {
    const reportPath = 'chaos-report.md';
    const timestamp = new Date().toISOString();

    let md = `# Identity Platform Chaos Agent Report\n\n`;
    md += `**Date:** ${timestamp}\n`;
    md += `**Total Scenarios:** ${results.length}\n`;
    const passed = results.filter(r => r.exitCode === 0).length;
    md += `**Passed:** ${passed}/${results.length}\n`;
    md += `**Status:** ${passed === results.length ? '✅ ALL PASSED' : '❌ FAILURES DETECTED'}\n\n`;

    md += `## Summary\n\n| Scenario | Status | Duration | Exit Code |\n|---|---|---|---|\n`;

    results.forEach(r => {
        const icon = r.exitCode === 0 ? '✅' : '❌';
        md += `| ${r.name} | ${icon} | ${r.duration.toFixed(1)}s | ${r.exitCode} |\n`;
    });

    md += `\n## Failure Logs\n`;

    const failures = results.filter(r => r.exitCode !== 0);

    if (failures.length === 0) {
        md += `*No failures to report.*`;
    } else {
        failures.forEach(r => {
            md += `### ❌ ${r.name}\n`;
            md += `**Command:** \`${r.command}\`\n\n`;
            // Simple ANSI code stripping could be good here but might be complex without a library.
            // We'll leave codes as raw output often handles them or Markdown validation doesn't care content-wise.
            // For cleaner markdown, we might want to strip them, but for raw debug log in MD code block, escape codes are usually harmless or rendered as junk.
            // Let's strip valid JSON-breaking chars if any.
            md += `\`\`\`text\n${r.output.slice(-3000)}\n\`\`\`\n\n`; // Last 3000 chars
            md += `*(Log truncated to last 3000 characters)*\n\n---\n`;
        });
    }

    fs.writeFileSync(reportPath, md);
    console.log(chalk.bold.magenta(`\n[Runner] Report generated: ${path.resolve(reportPath)}`));
}

async function main() {
    console.log(chalk.bold('🚀 Starting Chaos Agent Test Suite'));
    const results: ScenarioResult[] = [];

    for (const scenario of scenarios) {
        const result = await runScenario(scenario);
        results.push(result);
    }

    generateReport(results);

    const hasFailure = results.some(r => r.exitCode !== 0);
    process.exit(hasFailure ? 1 : 0);
}

main();
