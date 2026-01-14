import { Page } from 'playwright';
import { Graph } from '../core/graph';
import { AgentState } from '../core/types';
import { ActionExecutor } from '../core/executor';
import { agentState } from '../core/state';
import { UserProfile } from '../core/user-profiles';
import chalk from 'chalk';

export class ValidationEngine {
    private stateVisitCount: Map<string, number> = new Map();
    private consecutiveNoTransition: number = 0;
    private MAX_NO_TRANSITION = 3; // Exit if stuck for 3 iterations
    private MAX_STATE_VISITS = 2; // Visit each state max 2 times

    constructor(
        private page: Page,
        private graph: Graph,
        private executor: ActionExecutor
    ) {}

    async run(durationSeconds: number = 60): Promise<boolean> {
        console.log(chalk.cyan('[Validation] Starting deterministic validation...'));
        const endTime = Date.now() + durationSeconds * 1000;
        let iterationCount = 0;
        let success = false;

        // Simple loop: Identify current state -> Check transitions -> Execute "Happy Path" rule
        while (Date.now() < endTime) {
            iterationCount++;
            const currentUrl = this.page.url();
            const state = await this.graph.matchState(currentUrl, this.page);

            if (!state) {
                console.log(chalk.yellow(`[Validation] Unknown state: ${currentUrl}. Waiting...`));
                await this.page.waitForTimeout(2000);
                this.consecutiveNoTransition++;

                if (this.consecutiveNoTransition >= this.MAX_NO_TRANSITION) {
                    console.log(chalk.red('[Validation] Too many unknown states. Exiting validation.'));
                    break;
                }
                continue;
            }

            console.log(chalk.cyan(`[Validation] Current State: ${state.name} (iteration ${iterationCount})`));

            // Track state visits
            const visitCount = (this.stateVisitCount.get(state.id) || 0) + 1;
            this.stateVisitCount.set(state.id, visitCount);
            console.log(chalk.gray(`[Validation] State visit count: ${visitCount}`));

            // Get transitions from this state
            const transitions = this.graph.getPossibleTransitions(state.id);
            if (transitions.length === 0) {
                console.log(chalk.red('[Validation] Dead end (no transitions defined). Exiting validation.'));
                break;
            }

            // Find first transition where condition is met (Deterministic)
            let executed = false;
            for (const t of transitions) {
                console.log(`[Validation] Checking rule: ${t.trigger.description}`);

                try {
                    if (await t.trigger.condition(this.page)) {
                        console.log(chalk.green(`[Validation] Executing: ${t.trigger.description}`));
                        await t.trigger.action(this.page);
                        executed = true;
                        this.consecutiveNoTransition = 0; // Reset counter on successful execution
                        break; // Execute one step then re-evaluate state
                    }
                } catch (error) {
                    console.log(chalk.yellow(`[Validation] Error checking/executing rule: ${error}`));
                    // Continue to next rule
                }
            }

            if (!executed) {
                console.log(chalk.yellow('[Validation] No transition conditions met.'));
                this.consecutiveNoTransition++;

                // Check if we've been stuck too long
                if (this.consecutiveNoTransition >= this.MAX_NO_TRANSITION) {
                    console.log(chalk.yellow('[Validation] Stuck with no valid transitions. Attempting final transition (logout)...'));

                    // Try to execute the last transition (usually logout) to complete the cycle
                    const lastTransition = transitions[transitions.length - 1];
                    if (lastTransition) {
                        console.log(chalk.cyan(`[Validation] Forcing: ${lastTransition.trigger.description}`));
                        try {
                            await lastTransition.trigger.action(this.page);
                            this.consecutiveNoTransition = 0;
                            executed = true;
                        } catch (error) {
                            console.log(chalk.red(`[Validation] Failed to force transition: ${error}`));
                        }
                    }

                    if (!executed) {
                        console.log(chalk.red('[Validation] Unable to proceed. Exiting validation.'));
                        break;
                    }
                }

                await this.page.waitForTimeout(1000);
            }

            // Success criteria: We've successfully logged in, set up MFA, and navigated dashboard
            const hasVisitedLogin = this.stateVisitCount.has('login');
            const hasVisitedSetup = this.stateVisitCount.has('setup-secure');
            const hasVisitedDashboard = this.stateVisitCount.has('dashboard');
            const hasVisitedTOTP = this.stateVisitCount.has('totp-verify');

            // Recovery flow success criteria
            const hasVisitedRecoveryEmail = this.stateVisitCount.has('recovery-email');
            const hasVisitedRecoveryCode = this.stateVisitCount.has('recovery-code');
            const hasVisitedRecoveryPassword = this.stateVisitCount.has('recovery-password');

            // WebAuthn flow success criteria
            const hasVisitedWebAuthnRegister = this.stateVisitCount.has('webauthn-register');
            const hasVisitedWebAuthnVerify = this.stateVisitCount.has('webauthn-verify');

            // Check for WebAuthn registration + 2FA login cycle
            // Flow: login -> totp -> dashboard -> register webauthn -> logout -> login -> webauthn -> dashboard
            // This combines registration, logout, and re-login with WebAuthn as 2FA
            if (hasVisitedLogin && hasVisitedWebAuthnVerify && hasVisitedDashboard && hasVisitedTOTP && hasVisitedWebAuthnRegister) {
                const loginVisits = this.stateVisitCount.get('login') || 0;
                const dashboardVisits = this.stateVisitCount.get('dashboard') || 0;
                const registerVisits = this.stateVisitCount.get('webauthn-register') || 0;

                // Complex scenario: registration + multiple keys + 2FA login + deletions
                if (loginVisits >= 2 && dashboardVisits >= 3 && registerVisits >= 3) {
                    console.log(chalk.green('[Validation] ✅ Successfully completed comprehensive WebAuthn test!'));
                    console.log(chalk.gray(`[Validation] Registered ${registerVisits} keys, performed 2FA login cycle`));
                    console.log(chalk.gray(`[Validation] States visited: ${Array.from(this.stateVisitCount.keys()).join(', ')}`));
                    success = true;
                    break;
                }

                // Simpler scenario: registration + 2FA re-login
                if (loginVisits >= 2 && dashboardVisits >= 3) {
                    console.log(chalk.green('[Validation] ✅ Successfully completed WebAuthn registration + 2FA login!'));
                    console.log(chalk.gray(`[Validation] States visited: ${Array.from(this.stateVisitCount.keys()).join(', ')}`));
                    success = true;
                    break;
                }
            }

            // Check for WebAuthn-only login (no TOTP)
            // Flow: login -> webauthn -> dashboard (user has WebAuthn, no TOTP)
            if (hasVisitedLogin && hasVisitedWebAuthnVerify && hasVisitedDashboard && !hasVisitedTOTP) {
                const dashboardVisits = this.stateVisitCount.get('dashboard') || 0;
                if (dashboardVisits >= 2) {
                    console.log(chalk.green('[Validation] ✅ Successfully completed WebAuthn-only login!'));
                    console.log(chalk.green(`[Validation] States visited: ${Array.from(this.stateVisitCount.keys()).join(', ')}`));
                    success = true;
                    break;
                }
            }

            // Check for recovery flow completion
            if (hasVisitedRecoveryEmail && hasVisitedRecoveryCode && hasVisitedRecoveryPassword) {
                console.log(chalk.green('[Validation] ✅ Successfully completed password recovery flow!'));
                console.log(chalk.gray(`[Validation] States visited: ${Array.from(this.stateVisitCount.keys()).join(', ')}`));
                success = true;
                break;
            }

            // Check for basic WebAuthn registration (no re-login)
            if (hasVisitedLogin && hasVisitedTOTP && hasVisitedWebAuthnRegister && hasVisitedDashboard && !hasVisitedWebAuthnVerify) {
                const dashboardVisits = this.stateVisitCount.get('dashboard') || 0;
                const registerVisits = this.stateVisitCount.get('webauthn-register') || 0;
                const loginVisits = this.stateVisitCount.get('login') || 0;

                // WebAuthn Flow: Register -> Logout -> Login
                const currentProfile = agentState.get<UserProfile>('userProfile');

                if (currentProfile?.name === 'webauthn-flow') {
                    const webauthnVerifyVisits = this.stateVisitCount.get('webauthn-verify') || 0;

                    // Exit early to next iteration if we haven't even registered
                    if (registerVisits === 0) continue;

                    // Success criteria:
                    // 1. Registered at least one key
                    // 2. Logged out (login visits >= 2)
                    // 3. Re-authenticated using WebAuthn (visited webauthn-verify state)
                    // 4. Reached dashboard again (dashboard visits >= 2)
                    if (registerVisits >= 1 && loginVisits >= 2 && webauthnVerifyVisits >= 1 && dashboardVisits >= 2) {
                        console.log(chalk.green('[Validation] ✅ Successfully registered WebAuthn key, logged out, and re-authenticated with 2FA!'));
                        console.log(chalk.gray(`[Validation] Registered keys: ${agentState.get('registeredKeysCount')}`));
                        success = true;
                        break;
                    }

                    // Continue loop to allow re-login flow to happen
                    continue;
                } else {
                    // Standard multiple key registration test (old logic for other profiles)
                    if (registerVisits >= 3 && dashboardVisits >= 2) {
                        console.log(chalk.green('[Validation] ✅ Successfully registered multiple WebAuthn keys!'));
                        console.log(chalk.gray(`[Validation] Registered ${registerVisits} security keys`));
                        console.log(chalk.gray(`[Validation] States visited: ${Array.from(this.stateVisitCount.keys()).join(', ')}`));
                        success = true;
                        break;
                    }
                }

                // Single key registration test
                if (dashboardVisits >= 2 && currentProfile?.name !== 'webauthn-flow') {
                    console.log(chalk.green('[Validation] ✅ Successfully completed WebAuthn registration!'));
                    console.log(chalk.gray(`[Validation] States visited: ${Array.from(this.stateVisitCount.keys()).join(', ')}`));
                    success = true;
                    break;
                }
            }

            if (hasVisitedLogin && hasVisitedDashboard) {
                const dashboardVisits = this.stateVisitCount.get('dashboard') || 0;

                // Success if we've completed OIDC flow (login + setup + dashboard)
                if (hasVisitedSetup && dashboardVisits >= 2) {
                    console.log(chalk.green('[Validation] ✅ Successfully completed happy path validation!'));
                    console.log(chalk.gray(`[Validation] States visited: ${Array.from(this.stateVisitCount.keys()).join(', ')}`));
                    success = true;
                    break;
                }

                // Or success if we've done standalone login and explored dashboard (login + TOTP + dashboard many times)
                if (hasVisitedTOTP && dashboardVisits >= 5) {
                    console.log(chalk.green('[Validation] ✅ Successfully completed standalone login validation!'));
                    console.log(chalk.gray(`[Validation] States visited: ${Array.from(this.stateVisitCount.keys()).join(', ')}`));
                    success = true;
                    break;
                }
            }
        }

        console.log(chalk.cyan('[Validation] Finished.'));
        console.log(chalk.gray(`[Validation] Final state visits: ${JSON.stringify(Array.from(this.stateVisitCount.entries()))}`));

        if (!success) {
            console.log(chalk.red('[Validation] ❌ Validation failed: Success criteria not met within duration.'));
        }

        return success;
    }
}
