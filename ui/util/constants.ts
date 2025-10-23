import { UiNode } from "@ory/client";

// see https://www.ory.sh/docs/kratos/concepts/ui-messages
const ORY_LABEL_SECURITY_KEY_ADD = 1050012;
const ORY_LABEL_SECURITY_KEY_NAME_INPUT = 1050013;
const ORY_LABEL_SECURITY_KEY_REMOVE = 1050018;
const ORY_LABEL_BACKUP_CODE_CREATE = 1050008;
const ORY_LABEL_BACKUP_CODE_CONFIRM_TEXT = 1050010;
const ORY_LABEL_BACKUP_CODE_CONFIRM = 1050011;
const ORY_LABEL_BACKUP_CODE_VIEW = 1050007;
const ORY_LABEL_BACKUP_CODE_DEACTIVATE = 1050016;
const ORY_LABEL_USE_AUTHENTICATOR = 1010009;
const ORY_LABEL_USE_BACKUP_CODE = 1010010;
const ORY_LABEL_SIGN_IN_EMAIL_INPUT = 1070002;
const ORY_LABEL_SIGN_IN_WITH_PASSWORD = 1010022;
const ORY_LABEL_CONTINUE_PASSWORD_RESET = 1070009;
const ORY_LABEL_SIGN_IN_WITH_HARDWARE_KEY = 1010008;
export const ORY_LABEL_CONTINUE_IDENTIFIER_FIRST_LOGIN = 1070009;

type NodeWithLabel = UiNode & { meta: { label: object } };

export const isSecurityKeyAddBtn = (node: UiNode): node is NodeWithLabel =>
  node.meta.label?.id === ORY_LABEL_SECURITY_KEY_ADD;

export const isSecurityKeyNameInput = (node: UiNode): node is NodeWithLabel =>
  node.meta.label?.id === ORY_LABEL_SECURITY_KEY_NAME_INPUT;

export const isSecurityKeyRemoveBtn = (node: UiNode): node is NodeWithLabel =>
  node.meta.label?.id === ORY_LABEL_SECURITY_KEY_REMOVE;

export const isBackupCodeCreate = (node: UiNode): node is NodeWithLabel =>
  node.meta.label?.id === ORY_LABEL_BACKUP_CODE_CREATE;

export const isBackupCodeConfirmText = (node: UiNode): node is NodeWithLabel =>
  node.meta.label?.id === ORY_LABEL_BACKUP_CODE_CONFIRM_TEXT;

export const isBackupCodeConfirm = (node: UiNode): node is NodeWithLabel =>
  node.meta.label?.id === ORY_LABEL_BACKUP_CODE_CONFIRM;

export const isBackupCodeView = (node: UiNode): node is NodeWithLabel =>
  node.meta.label?.id === ORY_LABEL_BACKUP_CODE_VIEW;

export const isBackupCodeDeactivate = (node: UiNode): node is NodeWithLabel =>
  node.meta.label?.id === ORY_LABEL_BACKUP_CODE_DEACTIVATE;

export const isUseAuthenticator = (node: UiNode): node is NodeWithLabel =>
  node.meta.label?.id === ORY_LABEL_USE_AUTHENTICATOR;

export const isUseBackupCode = (node: UiNode): node is NodeWithLabel =>
  node.meta.label?.id === ORY_LABEL_USE_BACKUP_CODE;

export const isSignInEmailInput = (node: UiNode): node is NodeWithLabel =>
  node.meta.label?.id === ORY_LABEL_SIGN_IN_EMAIL_INPUT;

export const isSignInWithPassword = (node: UiNode): node is NodeWithLabel =>
  node.meta.label?.id === ORY_LABEL_SIGN_IN_WITH_PASSWORD;

export const isContinueWithPasswordReset = (
  node: UiNode,
): node is NodeWithLabel =>
  node.meta.label?.id === ORY_LABEL_CONTINUE_PASSWORD_RESET;

export const isSignInWithHardwareKey = (node: UiNode): node is NodeWithLabel =>
  node.meta.label?.id === ORY_LABEL_SIGN_IN_WITH_HARDWARE_KEY;

export const ORY_ERR_ACCOUNT_NOT_FOUND_OR_NO_LOGIN_METHOD = 4000037;
