import { authSetup } from './base';
import { assign } from 'xstate';

export const backupCodeMachine = authSetup.createMachine({
  id: 'backupCodeFlow',
  initial: 'idle',
  context: ({ input }) => ({
    userProfile: input.userProfile,
    registeredKeys: 0,
    url: input.url || '',
    page: undefined,
    backupCode: undefined
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
                    target: 'navigatingToBackupCodes' // Assume we are logged in if no MFA needed
                }
            ]
        }
    },
    verifyingTotp: {
        invoke: {
            src: 'verifyTotp',
            input: ({ context }) => ({ page: context.page }),
            onDone: 'navigatingToBackupCodes'
        }
    },
    navigatingToBackupCodes: {
        invoke: {
            src: 'navigateToBackupCodes',
            input: ({ context }) => ({ page: context.page }),
            onDone: 'generatingBackupCodes'
        }
    },
    generatingBackupCodes: {
        invoke: {
            src: 'generateBackupCodes',
            input: ({ context }) => ({ page: context.page }),
            onDone: {
                target: 'savingBackupCodes',
                actions: assign({
                    backupCode: ({ event }) => event.output
                })
            }
        }
    },
    savingBackupCodes: {
        invoke: {
            src: 'completeBackupCodesSetup',
            input: ({ context }) => ({ page: context.page }),
            onDone: 'loggingOut'
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
            onDone: 'verifyingBackupCode'
        }
    },
    verifyingBackupCode: {
        invoke: {
            src: 'enterBackupCode',
            input: ({ context }) => ({ page: context.page, code: context.backupCode || '' }),
            onDone: 'finalDashboard'
        }
    },
    finalDashboard: {
        type: 'final'
    }
  }
});
