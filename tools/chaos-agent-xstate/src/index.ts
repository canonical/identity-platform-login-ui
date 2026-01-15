import { chromium } from 'playwright';
import { createActor, fromPromise, AnyStateMachine } from 'xstate';
import { recoveryMachine } from './machines/recovery.machine';
import { webauthnMachine } from './machines/webauthn.machine';
import { totpMachine } from './machines/totp.machine';
import { backupCodeMachine } from './machines/backup-code.machine';
import { Driver } from './drivers/root';
import { Reporter } from './reporter';
import { UserProfile } from './types';
import { program } from 'commander';
import chalk from 'chalk';

import { KratosAdmin } from './utils/kratos-admin';

program
  .option('--profile <profile>', 'User profile to use', 'webauthn-flow')
  .option('--base-url <url>', 'Base URL', 'http://localhost')
  .option('--headed', 'Run in headed mode', false)
  .option('--video', 'Record video of the test run', false)
  .option('--scenario <scenario>', 'Scenario to run', 'webauthn-login')
  .option('--reset-user', 'Reset user WebAuthn credentials before run', false)
  .option('--username <email>', 'User email', 'webauthn-dynamic@example.com')
  .option('--password <password>', 'User password', 'Password123!')
  .option('--kratos-admin-url <url>', 'Kratos Admin URL', 'http://localhost:4434')
  .parse(process.argv);

const options = program.opts();

async function run() {
    console.log(chalk.green('XState Chaos Agent v2 Starting...'));
    const reporter = new Reporter();

    // 1. Reset User State if requested
    if (options.resetUser) {
        try {
            const admin = new KratosAdmin(options.kratosAdminUrl);
            await admin.resetUser(options.username);
        } catch (e) {
            console.error(chalk.red('FATAL: Failed to reset user. Aborting test run.'));
            process.exit(1);
        }
    }

    const browser = await chromium.launch({ headless: !options.headed });

    const contextOptions: any = {};
    if (options.video) {
        contextOptions.recordVideo = {
            dir: reporter.reportDir,
            size: { width: 1280, height: 720 }
        };
    }

    const context = await browser.newContext(contextOptions);
    const page = await context.newPage();

    // Wire up logs
    const logger = (msg: string) => reporter.logInfo(msg);
    reporter.setPage(page);
    const driver = new Driver(page, logger);

    const profile = {
        name: options.profile,
        email: options.username,
        password: options.password
    };

/*
    if (profile.name !== 'webauthn-flow') {
        console.log(chalk.red('Only "webauthn-flow" profile is strictly supported in this v2 prototype.'));
        // process.exit(1);
    }
*/

    const loginUrl = `${options.baseUrl}/ui/login`;

    const machines: Record<string, AnyStateMachine> = {
        'recovery-flow': recoveryMachine,
        'webauthn-flow': webauthnMachine,
        'totp-flow': totpMachine,
        'backup-code-flow': backupCodeMachine
    };

    const selectedMachine = machines[profile.name];
    if (!selectedMachine) {
        console.error(chalk.red(`FATAL: Unknown profile "${profile.name}". Available: ${Object.keys(machines).join(', ')}`));
        process.exit(1);
    }

    const actor = createActor(selectedMachine.provide({
        actors: {
            navigateToLogin: fromPromise(async ({ input }: { input: { page: any; url: string } }) => {
                reporter.logStep('Navigating to login');
                await driver.auth.navigateToLogin(input.url);
            }),
            performLogin: fromPromise(async ({ input }: { input: { page: any; email: string; password?: string } }) => {
                reporter.logStep('Performing Login');
                await driver.auth.performLogin({ email: input.email, password: input.password });
            }),
            analyzeCurrentState: fromPromise(async ({ input }) => {
                reporter.logStep('Analyzing State');
                return await driver.auth.analyzeCurrentState();
            }),
            skipTotpSetup: fromPromise(async ({ input }) => {
                reporter.logStep('Skipping TOTP Setup');
                await driver.mfa.skipTotpSetup();
            }),
            setupTotp: fromPromise(async ({ input }) => {
                reporter.logStep('Setting up TOTP');
                await driver.mfa.setupTotp();
            }),
            verifyTotp: fromPromise(async ({ input }) => {
                reporter.logStep('Verifying TOTP');
                await driver.mfa.verifyTotp();
            }),
            registerWebAuthn: fromPromise(async ({ input }) => {
                reporter.logStep('Registering WebAuthn');
                await driver.mfa.registerWebAuthn();
            }),
            performLogout: fromPromise(async ({ input }) => {
                reporter.logStep('Performing Logout');
                await driver.auth.performLogout();
            }),
            performWebAuthnLogin: fromPromise(async ({ input }) => {
                reporter.logStep('Performing WebAuthn Login');
                await driver.mfa.performWebAuthnLogin();

                reporter.logStep('Validating Active Session');
                await driver.auth.validateSession();
            }),
            startRecovery: fromPromise(async ({ input }) => {
                reporter.logStep('Starting Recovery Flow');
                await driver.recovery.clickForgotPassword();
            }),
            submitRecoveryEmail: fromPromise(async ({ input }: { input: { email: string; page: any } }) => {
                reporter.logStep('Submitting Recovery Email');
                await driver.recovery.submitRecoveryEmail(input.email);
            }),
            enterRecoveryCode: fromPromise(async ({ input }: { input: { email: string; page: any } }) => {
                reporter.logStep('Entering Recovery Code');
                await driver.recovery.enterRecoveryCode(input.email);
            }),
            setNewPassword: fromPromise(async ({ input }: { input: { password: string; page: any } }) => {
                reporter.logStep('Setting New Password');
                await driver.recovery.enterNewPassword(input.password);
            }),
            navigateToBackupCodes: fromPromise(async ({ input }: { input: { page: any }}) => {
                reporter.logStep('Navigating to Backup Codes');
                await driver.settings.navigateToBackupCodes();
            }),
            generateBackupCodes: fromPromise(async ({ input }: { input: { page: any } }) => {
                reporter.logStep('Generating Backup Codes');
                return await driver.settings.generateBackupCodes();
            }),
            completeBackupCodesSetup: fromPromise(async ({ input }: { input: { page: any } }) => {
                reporter.logStep('Completing Backup Codes Setup');
                await driver.settings.completeBackupCodesSetup();
            }),
            enterBackupCode: fromPromise(async ({ input }: { input: { code: string; page: any } }) => {
                reporter.logStep(`Entering Backup Code: ${input.code}`);
                await driver.mfa.enterBackupCode(input.code);
            })
        }
    }), {
        input: {
            userProfile: profile,
            url: loginUrl,
            registeredKeys: 0
        }
    });

    const cleanup = async () => {
        if (options.video) {
            // Video is saved when context is closed
            const video = page.video();
            if (video) {
                const videoPath = await video.path();
                reporter.setVideoPath(videoPath);
            }
        }
        await context.close();
        await browser.close();
        await reporter.generateReport();
    };

    actor.subscribe(async (snapshot) => {
        console.log(chalk.cyan(`[State Change] -> ${JSON.stringify(snapshot.value)}`));

        if (snapshot.status === 'done') {
            console.log(chalk.green('Workflow Completed!'));
            await cleanup();
            process.exit(0);
        }
        if (snapshot.status === 'error') {
            console.log(chalk.red('Workflow Failed!'));
            const err = snapshot.error as Error;
            await reporter.logFailure(err.message || 'Unknown error');
            await cleanup();
            process.exit(1);
        }
    });

    actor.start();
    actor.send({ type: 'START', page });
}

run().catch(console.error);
