import { BrowserContext, CDPSession } from 'playwright';

export class WebAuthnService {
    private session: CDPSession | null = null;
    private authenticatorId: string | null = null;

    constructor(private context: BrowserContext) {}

    /**
     * Enables the Virtual Authenticator environment.
     * Must be called before any WebAuthn actions.
     */
    async enable() {
        if (this.session) return;

        // Create CDP session to talk to Chrome DevTools Protocol
        // We use the first page's target or the browser target?
        // Playwright exposes newCDPSession on the context.
        const page = this.context.pages()[0];
        if (!page) throw new Error('No pages open to attach CDP session');
        
        this.session = await this.context.newCDPSession(page);
        
        await this.session.send('WebAuthn.enable');
        
        // Create a virtual authenticator (simulates a YubiKey/TouchID)
        const result = await this.session.send('WebAuthn.addVirtualAuthenticator', {
            options: {
                protocol: 'ctap2',
                transport: 'usb',
                hasResidentKey: true,
                hasUserVerification: true,
                isUserVerified: true,
                automaticPresenceSimulation: true 
                // ^ This is key: it auto-"touches" the key when requested!
            }
        });
        
        this.authenticatorId = result.authenticatorId;
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
        }
    }
}
