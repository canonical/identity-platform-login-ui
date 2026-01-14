import axios from 'axios';

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
        console.log(`[EmailService] Fetching recovery code for ${email} (max ${retries} attempts)`);
        if (initialCount !== undefined) {
            console.log(`[EmailService] Waiting for email count to exceed ${initialCount}`);
        }
        
        for (let i = 0; i < retries; i++) {
            try {
                // Mailslurper API: /mail returns all mails
                const response = await axios.get<MailslurperResponse>(`${this.baseUrl}/mail`, {
                    timeout: 5000
                });
                
                const mails = response.data.mailItems || [];
                console.log(`[EmailService] Attempt ${i + 1}/${retries}: Found ${mails.length} total emails`);
                
                // Filter emails for this recipient and sort by date descending
                const myMails = mails
                    .filter((m: MailItem) => m.toAddresses.includes(email))
                    .sort((a: MailItem, b: MailItem) => 
                        new Date(b.dateSent).getTime() - new Date(a.dateSent).getTime()
                    );

                console.log(`[EmailService] Found ${myMails.length} emails for ${email}`);

                // If we're waiting for a new email, check if count increased
                if (initialCount !== undefined && myMails.length <= initialCount) {
                    console.log(`[EmailService] Email count (${myMails.length}) hasn't increased yet, waiting...`);
                } else if (myMails.length > 0) {
                    const latest = myMails[0];
                    console.log(`[EmailService] Latest email subject: "${latest.subject}" sent at ${latest.dateSent}`);
                    
                    // Extract 6-8 digit code from email body
                    // Kratos typically sends 6-digit codes
                    const match = latest.body.match(/\b\d{6,8}\b/);
                    if (match) {
                        console.log(`[EmailService] ✅ Found code: ${match[0]}`);
                        return match[0];
                    } else {
                        console.log(`[EmailService] No code found in email body`);
                    }
                }
            } catch (error) {
                if (axios.isAxiosError(error)) {
                    console.error(`[EmailService] HTTP error: ${error.message}`);
                    if (error.code === 'ECONNREFUSED') {
                        console.error(`[EmailService] ⚠️  Cannot connect to Mailslurper at ${this.baseUrl}`);
                        console.error(`[EmailService] Make sure Mailslurper is running (docker compose up mailslurper)`);
                    }
                } else {
                    console.error(`[EmailService] Failed to fetch mails: ${error}`);
                }
            }
            
            // Wait before retry (unless it's the last attempt)
            if (i < retries - 1) {
                console.log(`[EmailService] Waiting ${delayMs}ms before retry...`);
                await new Promise(r => setTimeout(r, delayMs));
            }
        }
        
        console.error(`[EmailService] ❌ Failed to retrieve code after ${retries} attempts`);
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
            console.error(`[EmailService] Failed to get email count: ${error}`);
            return 0;
        }
    }

    /**
     * Get the full email content for debugging
     */
    async getLatestEmail(email: string): Promise<MailItem | null> {
        try {
            const response = await axios.get<MailslurperResponse>(`${this.baseUrl}/mail`, {
                timeout: 5000
            });
            const mails = response.data.mailItems || [];
            
            const myMails = mails
                .filter((m: MailItem) => m.toAddresses.includes(email))
                .sort((a: MailItem, b: MailItem) => 
                    new Date(b.dateSent).getTime() - new Date(a.dateSent).getTime()
                );

            return myMails.length > 0 ? myMails[0] : null;
        } catch (error) {
            console.error(`[EmailService] Failed to fetch email: ${error}`);
            return null;
        }
    }
}
