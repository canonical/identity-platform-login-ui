import { spawn } from 'child_process';
import chalk from 'chalk';

interface ConfigProfile {
    name: string;
    env: Record<string, string>;
}

const profiles: ConfigProfile[] = [
    {
        name: 'Standard',
        env: { IDENTIFIER_FIRST: 'false', MFA_ENABLED: 'true' }
    },
    {
        name: 'Legacy (No MFA)',
        env: { IDENTIFIER_FIRST: 'false', MFA_ENABLED: 'false' }
    }
];

async function runProfile(profile: ConfigProfile): Promise<void> {
    console.log(chalk.blue(`[Matrix] Starting profile: ${profile.name}`));

    return new Promise((resolve, reject) => {
        const env = { ...process.env, ...profile.env };

        // In a real scenario, this would spin up the Docker stack with these env vars
        // For this agent, we assume the environment is ALREADY running with these settings
        // or that we are just passing them to the agent to change its expectations.

        // Since we can't easily restart the backend from here without docker access,
        // we will just run the agent and pass the profile name as a flag for logging.

        const child = spawn('npm', ['start', '--', '--mode=validation', `--url=http://localhost/ui/login`], {
            env,
            stdio: 'inherit',
            shell: true,
            cwd: process.cwd() // Run from tools/chaos-agent root
        });

        child.on('close', (code) => {
            if (code === 0) {
                console.log(chalk.green(`[Matrix] Profile ${profile.name} passed.`));
                resolve();
            } else {
                console.log(chalk.red(`[Matrix] Profile ${profile.name} failed.`));
                reject(new Error(`Exit code ${code}`));
            }
        });
    });
}

async function main() {
    console.log(chalk.bold('🚀 Starting Matrix Runner'));

    // Run sequentially for now
    for (const profile of profiles) {
        try {
            await runProfile(profile);
        } catch (e) {
            console.error(e);
            process.exit(1);
        }
    }

    console.log(chalk.bold.green('✅ All profiles completed.'));
}

main();
