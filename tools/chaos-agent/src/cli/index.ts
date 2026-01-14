#!/usr/bin/env node
import { Command } from 'commander';
import chalk from 'chalk';
import { chromium } from 'playwright';
import { Graph } from '../core/graph';
import { ActionExecutor } from '../core/executor';
import { DOMScanner } from '../core/scanner';
import { Reporter } from '../core/reporter';
import { ValidationEngine } from '../heuristics/validation';
import { ExplorationEngine } from '../heuristics/exploration';
import { WebAuthnService } from '../core/services/webauthn';
import { getUserProfile, listUserProfiles, UserProfile } from '../core/user-profiles';
import { agentState } from '../core/state';

const program = new Command();

program
  .name('chaos-agent')
  .description('AI-driven Chaos Agent for Identity Platform')
  .version('0.1.0')
  .option('-m, --mode <mode>', 'Mode: validation or exploration', 'validation')
  .option('-u, --url <url>', 'Target URL', 'http://127.0.0.1:4446/')
  .option('-d, --duration <seconds>', 'Duration in seconds', '60')
  .option('-o, --output <dir>', 'Output directory', 'test-results')
  .option('--headless', 'Run browser in headless mode', true)
  .option('--headed', 'Run browser in headed mode (shows browser UI)')
  .option('-p, --profile <profile>', 'User profile to use (basic, totp-user, webauthn-user, full-security)')
  .option('--list-profiles', 'List available user profiles')
  .action(async (options) => {
    if (options.listProfiles) {
        listUserProfiles();
        process.exit(0);
    }

    const reporter = new Reporter(options.output);
    reporter.logInfo(`Starting Chaos Agent in ${options.mode} mode`);
    reporter.logInfo(`Target: ${options.url}`);

    // Get user profile
    let userProfile: UserProfile | undefined;
    if (options.profile) {
        try {
            userProfile = getUserProfile(options.profile);

            reporter.logInfo(`Using user profile: ${userProfile.name} (${userProfile.email})`);
            reporter.logInfo(`  TOTP: ${userProfile.hasTotp ? '✓' : '✗'}, WebAuthn: ${userProfile.hasWebAuthn ? '✓' : '✗'}, Backup Codes: ${userProfile.hasBackupCodes ? '✓' : '✗'}`);

            // Store profile in agent state
            agentState.set('userProfile', userProfile);
            if (userProfile.totpSecret) {
                agentState.set('totpSecret', userProfile.totpSecret);
            }
        } catch (error) {
            reporter.logError(`Invalid user profile: ${error}`);
            listUserProfiles();
            process.exit(1);
        }
    }

    // Launch browser (scaffolding T015)
    const headless = options.headed ? false : options.headless;
    reporter.logInfo(`Browser mode: ${headless ? 'headless' : 'headed'}`);
    const browser = await chromium.launch({ headless });
    const context = await browser.newContext();
    await reporter.startTracing(context);
    const page = await context.newPage();

    // Initialize WebAuthn service early
    const webAuthnService = new WebAuthnService(context);

    // Enable WebAuthn if user profile has it
    if (userProfile && userProfile.hasWebAuthn) {
        reporter.logInfo('Enabling WebAuthn virtual authenticator for profile...');
        await webAuthnService.enable();
    }

    try {
        await page.goto(options.url);
        reporter.logInfo(`Navigated to ${await page.title()}`);

        // Initialize Core Components
        const graph = new Graph();
        const executor = new ActionExecutor(page);
        const scanner = new DOMScanner(page);

        // Load Rules (TODO: Dynamic loading based on config?)
        const { loginRules } = await import('../heuristics/rules/login');
        const { recoveryRules } = await import('../heuristics/rules/recovery');
        const { webauthnRules } = await import('../heuristics/rules/webauthn');
        const { settingsRules } = await import('../heuristics/rules/settings');
        const { negativeRules } = await import('../heuristics/rules/negative');
        const { oidcRules } = await import('../heuristics/rules/oidc');
        const { dashboardRules } = await import('../heuristics/rules/dashboard');
        const { setupRules } = await import('../heuristics/rules/setup');

        loginRules(graph);
        recoveryRules(graph);
        settingsRules(graph);
        negativeRules(graph);
        oidcRules(graph);
        dashboardRules(graph);
        webauthnRules(graph, webAuthnService);
        setupRules(graph);

        if (options.mode === 'validation') {
            const engine = new ValidationEngine(page, graph, executor);
            const success = await engine.run(parseInt(options.duration));
            if (!success) {
                throw new Error('Validation failed: Success criteria not met');
            }
        } else {
            const engine = new ExplorationEngine(page, scanner, executor);
            await engine.run(parseInt(options.duration));
        }

    } catch (error) {
        reporter.logError('Error during execution:', error);
        await reporter.captureFailure(page, `crash-${options.mode}`);
        process.exitCode = 1;
    } finally {
        await reporter.stopTracing(context, `run-${options.mode}`);
        await browser.close();
        reporter.logSuccess('Agent finished.');
    }
  });

program.parse();
