import axios from 'axios';
import chalk from 'chalk';

interface MailItem {
    toAddresses: string[];
    dateSent: string;
    body: string;
    subject: string;
}

interface MailslurperResponse {
    mailItems: MailItem[];
}

export class EmailService {
    private baseUrl: string;

    constructor(baseUrl: string = 'http://localhost:4437') {
        this.baseUrl = baseUrl;
    }

    /**
     * Polls Mailslurper for a NEW email sent to the given address
     * and extracts a verification code (assumes 6-8 digit code).
     * Waits for an email that arrives after initialCount.
     */
    async getLatestCode(email: string, retries = 10, delayMs = 1000, initialCount?: number): Promise<string | null> {
        console.log(chalk.gray(`[EmailService] Fetching recovery code for ${email} (max ${retries} attempts)`));
        if (initialCount !== undefined) {
            console.log(chalk.gray(`[EmailService] Waiting for email count to exceed ${initialCount}`));
        }

        for (let i = 0; i < retries; i++) {
            try {
                // Mailslurper API: /mail returns all mails
                const response = await axios.get<MailslurperResponse>(`${this.baseUrl}/mail`, {
                    timeout: 5000
                });

                const mails = response.data.mailItems || [];

                // Filter emails for this recipient and sort by date descending
                const myMails = mails
                    .filter((m: MailItem) => m.toAddresses.includes(email))
                    .sort((a: MailItem, b: MailItem) =>
                        new Date(b.dateSent).getTime() - new Date(a.dateSent).getTime()
                    );

                // If we're waiting for a new email, check if count increased
                if (initialCount !== undefined && myMails.length <= initialCount) {
                    // console.log(chalk.gray(`[EmailService] Email count (${myMails.length}) hasn't increased yet, waiting...`));
                } else if (myMails.length > 0) {
                    const latest = myMails[0];
                    console.log(chalk.green(`[EmailService] Latest email subject: "${latest.subject}" sent at ${latest.dateSent}`));

                    // Extract 6-8 digit code from email body
                    // Kratos typically sends 6-digit codes
                    const match = latest.body.match(/\b\d{6,8}\b/);
                    if (match) {
                        console.log(chalk.green(`[EmailService] ✅ Found code: ${match[0]}`));
                        return match[0];
                    } else {
                        console.log(chalk.yellow(`[EmailService] No code found in email body`));
                    }
                }
            } catch (error) {
                if (axios.isAxiosError(error)) {
                    if (error.code === 'ECONNREFUSED') {
                        console.error(chalk.red(`[EmailService] ⚠️  Cannot connect to Mailslurper at ${this.baseUrl}`));
                    }
                } else {
                    console.error(chalk.red(`[EmailService] Failed to fetch mails: ${error}`));
                }
            }

            // Wait before retry (unless it's the last attempt)
            if (i < retries - 1) {
                await new Promise(r => setTimeout(r, delayMs));
            }
        }

        console.error(chalk.red(`[EmailService] ❌ Failed to retrieve code after ${retries} attempts`));
        return null;
    }

    /**
     * Get the current count of emails for a specific address
     */
    async getEmailCount(email: string): Promise<number> {
        try {
            const response = await axios.get<MailslurperResponse>(`${this.baseUrl}/mail`, {
                timeout: 5000
            });
            const mails = response.data.mailItems || [];
            const myMails = mails.filter((m: MailItem) => m.toAddresses.includes(email));
            return myMails.length;
        } catch (error) {
             console.error(chalk.yellow(`[EmailService] Failed to count mails: ${error}`));
             return 0;
        }
    }
}
