import { Page } from 'playwright';

export interface UserProfile {
    name: string;
    email: string;
    password?: string;
    totpSecret?: string;
    hasWebAuthn?: boolean;
    hasTOTP?: boolean;
}

export interface ChaosContext {
    page?: Page; // Playwright page reference (can be undefined initially)
    userProfile: UserProfile;
    registeredKeys: number;
    url: string;
    backupCode?: string;
}

export type ChaosEvent =
    | { type: 'START', page: Page }
    | { type: 'LOGIN_SUBMIT' }
    | { type: 'MFA_REQUIRED' }
    | { type: 'SKIP_TOTP_SETUP' }
    | { type: 'REGISTER_WEBAUTHN' }
    | { type: 'LOGOUT' }
    | { type: 'WEBAUTHN_VERIFY' }
    | { type: 'DASHBOARD_LOADED' }
    | { type: 'ERROR', message: string };
