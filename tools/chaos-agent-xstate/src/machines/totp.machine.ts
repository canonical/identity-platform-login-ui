import { authSetup } from './base';
import { assign } from 'xstate';

export const totpMachine = authSetup.createMachine({
  id: 'totpFlow',
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
                    guard: ({ event }) => event.output === 'totp_verify',
                    target: 'verifyingTotp'
                },
                {
                    // If setup is needed
                     guard: ({ event }) => event.output === 'setup_totp',
                     // Skip for now, assume pre-configured user as per scenario requirements
                     // or handle setup here
                     target: 'settingUpTotp'
                },
                {
                    target: 'dashboard'
                }
            ]
        }
    },
    settingUpTotp: {
        invoke: {
            src: 'setupTotp',
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
            onDone: 'verifyingTotp'
        }
    },
    verifyingTotp: {
        invoke: {
            src: 'verifyTotp',
            input: ({ context }) => ({ page: context.page }),
            onDone: 'dashboard'
        }
    },
    dashboard: {
        type: 'final'
    }
  }
});
