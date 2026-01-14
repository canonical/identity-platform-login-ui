import { BrowserContext, CDPSession } from 'playwright';
const fs = require('fs');

export class WebAuthnService {
    private session: CDPSession | null = null;
    private authenticatorId: string | null = null;

    constructor(private context: BrowserContext) {}

    /**
     * Enables the Virtual Authenticator environment.
     * Must be called before any WebAuthn actions.
     */
    async enable() {
        if (this.session) {
            console.log('[WebAuthnService] Already enabled');
            return;
        }

        const page = this.context.pages()[0];
        if (!page) throw new Error('No pages open to attach CDP session');

        this.session = await this.context.newCDPSession(page);

        await this.session.send('WebAuthn.enable');
        console.log('[WebAuthnService] WebAuthn protocol enabled');

        const result = await this.session.send('WebAuthn.addVirtualAuthenticator', {
            options: {
                protocol: 'ctap2',
                transport: 'usb',
                hasResidentKey: true,
                hasUserVerification: true,
                isUserVerified: true,
                automaticPresenceSimulation: true
            }
        });
        // 3. Load the key data from your local JSON file
        const keyData = JSON.parse(fs.readFileSync('webauthn-authenticators.json', 'utf8'));

        // 4. Inject the credential into the virtual authenticator
        await this.session.send('WebAuthn.addCredential', {
            authenticatorId: result.authenticatorId,
            credential: keyData.credentials[0] // assuming the file contains an array of credentials,
        });

        this.authenticatorId = result.authenticatorId;
        console.log(`[WebAuthnService] ✅ Virtual Authenticator created: ${this.authenticatorId}`);
    }

    /**
     * Adds a credential to the virtual authenticator.
     * This simulates a previously registered security key.
     */
    async addCredential(credentialId: string, privateKey: string, rpId: string, userHandle: string, signCount: number = 0) {
        if (!this.session || !this.authenticatorId) {
            throw new Error('WebAuthn service not enabled. Call enable() first.');
        }

        await this.session.send('WebAuthn.addCredential', {
            authenticatorId: this.authenticatorId,
            credential: {
                credentialId,
                isResidentCredential: true,
                rpId,
                privateKey,
                userHandle,
                signCount
            }
        });

        console.log(`[WebAuthnService] ✅ Credential added to virtual authenticator`);
    }

    /**
     * Disables WebAuthn environment and removes authenticators.
     */
    async disable() {
        if (this.session && this.authenticatorId) {
            await this.session.send('WebAuthn.removeVirtualAuthenticator', {
                authenticatorId: this.authenticatorId
            });
            await this.session.send('WebAuthn.disable');
            await this.session.detach();
            this.session = null;
            this.authenticatorId = null;
            console.log('[WebAuthnService] Virtual Authenticator removed');
        }
    }
}
