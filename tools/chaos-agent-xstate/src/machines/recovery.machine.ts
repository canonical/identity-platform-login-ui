import { authSetup } from './base';
import { assign } from 'xstate';

export const recoveryMachine = authSetup.createMachine({
  id: 'recoveryFlow',
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
        onDone: 'startingRecovery'
      }
    },
    startingRecovery: {
        description: 'Clicking Forgot Password',
        invoke: {
            src: 'startRecovery',
            input: ({ context }) => ({ page: context.page }),
            onDone: 'enteringEmail'
        }
    },
    enteringEmail: {
        description: 'Entering recovery email',
        invoke: {
            src: 'submitRecoveryEmail',
            input: ({ context }) => ({ page: context.page, email: context.userProfile.email }),
            onDone: 'enteringCode'
        }
    },
    enteringCode: {
        description: 'Fetching and entering recovery code',
        invoke: {
            src: 'enterRecoveryCode',
            input: ({ context }) => ({ page: context.page, email: context.userProfile.email }),
            onDone: 'settingPassword'
        }
    },
    settingPassword: {
        description: 'Setting new password',
        invoke: {
            src: 'setNewPassword',
            input: ({ context }) => ({
                page: context.page,
                password: context.userProfile.password || ''
            }),
            onDone: 'dashboard'
        }
    },
    dashboard: {
        type: 'final'
    }
  }
});
