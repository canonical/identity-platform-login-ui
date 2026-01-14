import * as OTPAuth from 'otpauth';

export class TOTPService {
    /**
     * Generates a current TOTP token for the given secret.
     * Secret should be a base32 string.
     */
    static generate(secret: string): string {
        const totp = new OTPAuth.TOTP({
            issuer: 'IdentityPlatform',
            label: 'ChaosAgent',
            algorithm: 'SHA1',
            digits: 6,
            period: 30,
            secret: OTPAuth.Secret.fromBase32(secret)
        });
        return totp.generate();
    }
}
