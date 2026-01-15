import axios, { AxiosInstance } from 'axios';
import chalk from 'chalk';

export class KratosAdmin {
    private client: AxiosInstance;

    constructor(baseUrl: string) {
        this.client = axios.create({
            baseURL: baseUrl,
            timeout: 5000,
            headers: {
                'Content-Type': 'application/json'
            }
        });
    }

    async getIdentityId(email: string): Promise<string | null> {
        try {
            const res = await this.client.get('/admin/identities', {
                params: {
                    credentials_identifier: email
                }
            });
            if (res.data && res.data.length > 0) {
                return res.data[0].id;
            }
            return null;
        } catch (error) {
            console.error(chalk.red(`Failed to fetch identity for ${email}: ${error}`));
            return null;
        }
    }

async resetUser(email: string): Promise<void> {
        console.log(chalk.blue(`[KratosAdmin] Resetting credentials for ${email}...`));
        const id = await this.getIdentityId(email);

        if (!id) {
            console.log(chalk.yellow(`[KratosAdmin] User ${email} not found. Skipping reset.`));
            return;
        }

        try {
            // Use specific DELETE endpoints for credentials
            // This is cleaner than PUT and doesn't require knowing the password hash

            // 1. Delete WebAuthn
            try {
                await this.client.delete(`/admin/identities/${id}/credentials/webauthn`);
                console.log(chalk.green(`[KratosAdmin] Cleared WebAuthn.`));
            } catch (e: any) {
                if (e.response?.status !== 404) {
                    console.log(chalk.yellow(`[KratosAdmin] Warning deleting WebAuthn: ${e.message}`));
                }
            }

            // 2. Delete TOTP
            try {
                await this.client.delete(`/admin/identities/${id}/credentials/totp`);
                console.log(chalk.green(`[KratosAdmin] Cleared TOTP.`));
            } catch (e: any) {
                if (e.response?.status !== 404) {
                    console.log(chalk.yellow(`[KratosAdmin] Warning deleting TOTP: ${e.message}`));
                }
            }

        } catch (error) {
            console.error(chalk.red(`[KratosAdmin] Fatal error during reset: ${error}`));
            throw error;
        }
    }
}
