import { authSetup } from './base';
import { assign } from 'xstate';

export const webauthnMachine = authSetup.createMachine({
  id: 'webauthnFlow',
  initial: 'idle',
  context: ({ input }) => ({
    userProfile: input.userProfile,
    registeredKeys: 0,
    url: input.url || '',
    page: undefined
  }),
  states: {
    idle: {
      on: {
        START: {
          target: 'navigatingToLogin',
          actions: assign(({ event }) => ({ page: event.page }))
        }
      }
    },
    navigatingToLogin: {
      invoke: {
        src: 'navigateToLogin',
        input: ({ context }) => ({ page: context.page, url: context.url }),
        onDone: 'loggingIn'
      }
    },
    loggingIn: {
      invoke: {
        src: 'performLogin',
        input: ({ context }) => ({
            page: context.page,
            email: context.userProfile.email,
            password: context.userProfile.password
        }),
        onDone: 'analyzingState'
      }
    },
    analyzingState: {
        invoke: {
            src: 'analyzeCurrentState',
            input: ({ context }) => ({ page: context.page }),
            onDone: [
                {
                    guard: ({ event }) => event.output === 'setup_totp',
                    target: 'skippingTotp'
                },
                {
                    guard: ({ event }) => event.output === 'setup_passkey',
                    target: 'registeringWebAuthn'
                },
                {
                    guard: ({ event }) => event.output === 'webauthn_verify',
                    target: 'verifyingWebAuthn'
                },
                {
                    target: 'dashboard'
                }
            ]
        }
    },
    skippingTotp: {
        invoke: {
            src: 'skipTotpSetup',
            input: ({ context }) => ({ page: context.page }),
            onDone: 'registeringWebAuthn'
        }
    },
    registeringWebAuthn: {
        invoke: {
            src: 'registerWebAuthn',
            input: ({ context }) => ({ page: context.page }),
            onDone: {
                target: 'loggingOut',
                actions: assign({ registeredKeys: 1 })
            }
        }
    },
    loggingOut: {
        invoke: {
            src: 'performLogout',
            input: ({ context }) => ({ page: context.page }),
            onDone: 'reLoggingIn'
        }
    },
    reLoggingIn: {
        invoke: {
            src: 'performLogin',
            input: ({ context }) => ({
                page: context.page,
                email: context.userProfile.email,
                password: context.userProfile.password
            }),
            onDone: 'verifyingWebAuthn'
        }
    },
    verifyingWebAuthn: {
        invoke: {
            src: 'performWebAuthnLogin',
            input: ({ context }) => ({ page: context.page }),
            onDone: 'dashboard'
        }
    },
    dashboard: {
        type: 'final'
    }
  }
});
