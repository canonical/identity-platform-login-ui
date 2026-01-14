// Copyright 2025 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0

import * as fs from 'fs';
import * as path from 'path';

export interface UserProfile {
    name: string;
    email: string;
    password: string;
    totpSecret?: string;
    hasTotp: boolean;
    hasWebAuthn: boolean;
    hasBackupCodes: boolean;
    description: string;
}

export const USER_PROFILES: Record<string, UserProfile> = {
    'basic': {
        name: 'basic',
        email: 'test@example.com',
        password: 'test',
        hasTotp: false,
        hasWebAuthn: false,
        hasBackupCodes: false,
        description: 'Basic user without 2FA (MFA not enforced)'
    },
    'totp-user': {
        name: 'totp-user',
        email: 'login-test@example.com',
        password: 'Test1234!',
        hasTotp: true,
        hasWebAuthn: false,
        hasBackupCodes: false,
        description: 'User with TOTP 2FA configured'
    },
    'webauthn-user': {
        name: 'webauthn-user',
        email: 'webauthn-tester@example.com',
        password: 'Password123!',
        hasTotp: false,  // TOTP removed to test WebAuthn-only
        hasWebAuthn: true,
        hasBackupCodes: false,
        description: 'User with WebAuthn only (TOTP removed)'
    },
    'webauthn-flow': {
        name: 'webauthn-flow',
        email: 'webauthn-dynamic@example.com',
        password: 'Password123!',
        hasTotp: false,
        hasWebAuthn: false, // Starts false, registers in test
        hasBackupCodes: false,
        description: 'User for testing WebAuthn registration and subsequent login'
    },
    'full-security': {
        name: 'full-security',
        email: 'full-security@example.com',
        password: 'SecurePass123!',
        hasTotp: true,
        hasWebAuthn: true,
        hasBackupCodes: true,
        description: 'User with all security features (TOTP, WebAuthn, Backup Codes)'
    }
};

export function getUserProfile(profileName: string): UserProfile {
    // Try to load dynamic state from .chaos-agent-state.json
    try {
        // We look for the state file in the current working directory
        const statePath = path.resolve(process.cwd(), '.chaos-agent-state.json');

        if (fs.existsSync(statePath)) {
            const state = JSON.parse(fs.readFileSync(statePath, 'utf-8'));

            // 1. Merge 'users' state (Preferred source of truth)
            if (state.users) {
                Object.values(USER_PROFILES).forEach(profile => {
                    const userState = state.users[profile.email];
                    if (userState) {
                        // Merge relevant fields from persistent state
                        if (userState.totpSecret) profile.totpSecret = userState.totpSecret;
                        if (typeof userState.hasTotp === 'boolean') profile.hasTotp = userState.hasTotp;
                        if (typeof userState.hasWebAuthn === 'boolean') profile.hasWebAuthn = userState.hasWebAuthn;
                    }
                });
            }

            // 2. Legacy: Inject TOTP secrets from 'secrets' map
            if (state.secrets) {
                Object.values(USER_PROFILES).forEach(profile => {
                    if (state.secrets[profile.email]) {
                        profile.totpSecret = state.secrets[profile.email];
                    }
                });
            }

            // 3. Legacy/Fallback: Inject TOTP secret for totp-user if available
            if (state.totpSecret && USER_PROFILES['totp-user'] && !USER_PROFILES['totp-user'].totpSecret) {
                USER_PROFILES['totp-user'].totpSecret = state.totpSecret;
            }
        }
    } catch (e) {
        // Ignore errors if file doesn't exist or is invalid
        // console.warn('Note: Could not load .chaos-agent-state.json', e);
    }

    const profile = USER_PROFILES[profileName];
    if (!profile) {
        throw new Error(`Unknown user profile: ${profileName}. Available profiles: ${Object.keys(USER_PROFILES).join(', ')}`);
    }
    return profile;
}

export function listUserProfiles(): void {
    console.log('\nAvailable User Profiles:');
    console.log('========================\n');
    Object.entries(USER_PROFILES).forEach(([key, profile]) => {
        console.log(`${key}:`);
        console.log(`  Email: ${profile.email}`);
        console.log(`  TOTP: ${profile.hasTotp ? '✓' : '✗'}`);
        console.log(`  WebAuthn: ${profile.hasWebAuthn ? '✓' : '✗'}`);
        console.log(`  Backup Codes: ${profile.hasBackupCodes ? '✓' : '✗'}`);
        console.log(`  Description: ${profile.description}\n`);
    });
}
