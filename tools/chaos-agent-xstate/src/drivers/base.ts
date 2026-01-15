import { Page, CDPSession } from 'playwright';

export interface DriverState {
    currentUserEmail: string | null;
    cdpSession: CDPSession | null;
    authenticatorId: string | null;
    savedTotpSecret: string | null;
    savedCredential: any | null; // WebAuthn credential
}

export class BaseDriver {
    constructor(
        protected page: Page,
        protected state: DriverState,
        protected logger: (msg: string) => void
    ) {}

    protected log(msg: string) {
        this.logger(msg);
    }
}
