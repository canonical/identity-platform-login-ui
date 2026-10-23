import { UiNode, UiNodeInputAttributes, UiNodeMeta, UiText } from "@ory/client";

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
const ORY_LABEL_SIGN_IN_EMAIL_INPUT = 1070002; // this is wrong since it's used also for "Full name" and "Email verified"
const ORY_LABEL_SIGN_IN_WITH_PASSWORD = 1010022;
const ORY_LABEL_CONTINUE_PASSWORD_RESET = 1070009;
const ORY_LABEL_SIGN_IN_WITH_HARDWARE_KEY = 1010008;
const ORY_LABEL_RESEND_VERIFICATION_CODE = 1070008;
const ORY_LABEL_VERIFICATION_CODE_INPUT = 1070011;

const ORY_LABEL_REGISTER_EMAIL_INPUT = 1070002;
const ORY_LABEL_REGISTER_EMAIL_SUBMIT = 1040001;
const ORY_LABEL_REGISTER_PASSWORD_INPUT = 1070001;
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

export const isResendVerificationCode = (node: UiNode): node is NodeWithLabel =>
  node.meta.label?.id === ORY_LABEL_RESEND_VERIFICATION_CODE;

export const isVerificationCodeInput = (node: UiNode): node is NodeWithLabel =>
  node.meta.label?.id === ORY_LABEL_VERIFICATION_CODE_INPUT;

export const isRegisterEmailInput = (node: UiNode): node is NodeWithLabel =>
  node.meta.label?.id === ORY_LABEL_REGISTER_EMAIL_INPUT;

export const isRegisterEmailSubmit = (node: UiNode): node is NodeWithLabel =>
  node.meta.label?.id === ORY_LABEL_REGISTER_EMAIL_SUBMIT;

export const isRegisterPasswordInput = (node: UiNode): node is NodeWithLabel =>
  node.meta.label?.id === ORY_LABEL_REGISTER_PASSWORD_INPUT;

export const ORY_ERR_ACCOUNT_NOT_FOUND_OR_NO_LOGIN_METHOD = 4000037;

// Kratos errors the backend answers in its own words: the keys of uiErrorText
// in pkg/kratos/ui_errors.go. The page shows that answer where the error
// happened, so they are not shown again as flow-level messages.
const ORY_ERR_ANSWERED_BY_BACKEND = new Set([
  4000006, // IncorrectCredentials
  4000037, // IncorrectAccountIdentifier
  4000010, // AddressNotVerified
  4010011, // IdentityDisabled
  4000002, // PropertyMissing
  4000003, // NotEnoughCharacters
  4000017, // TooManyCharacters
  4000032, // PasswordTooShort
  4000033, // PasswordTooLong
  4000005, // PasswordPolicyViolation
  4000034, // PasswordBreached
  4000031, // PasswordIdentifierSimilarity
  4000039, // PasswordSameAsOld
  4000008, // InvalidAuthCode
  4000011, // MissingTOTPSetup
  4000013, // MissingSecurityKey
  4000015, // MissingSecurityKeySetup
  4000012, // BackupCodeAlreadyUsed
  4000016, // InvalidBackupCode
  4000014, // MissingBackupCodesSetup
  4000007, // DuplicateIdentifier
  4000027, // DuplicateIdentifierOIDCLink
  4000028, // DuplicateIdentifierWithHints
  4060006, // InvalidRecoveryCode
]);

export const isErrorAnsweredByBackend = (message: UiText): boolean =>
  ORY_ERR_ANSWERED_BY_BACKEND.has(message.id);

// Node groups and fields the backend adds to a flow for the tenant list and
// the company sign-ins. They are not part of the Kratos client types.
const TENANT_NODE_GROUP = "tenant";
const SSO_NODE_GROUP = "sso";
const TENANT_FIELD = "sso_tenant";
const SSO_LINK_NODE_PREFIX = "sso_link_";
export const SSO_UNLINK_FIELD = "sso_unlink";
export const SSO_UNLINK_LABEL_PREFIX = "Unlink ";
export const TENANT_INVITATION_LABEL = " — invitation";

export const isTenantNode = (node: UiNode): boolean =>
  (node.group as string) === TENANT_NODE_GROUP;

export const isSsoNode = (node: UiNode): boolean =>
  (node.group as string) === SSO_NODE_GROUP;

export const isTenantChoice = (node: UiNode): boolean =>
  isTenantNode(node) &&
  (node.attributes as UiNodeInputAttributes).name === TENANT_FIELD;

export const isSsoUnlinkBtn = (node: UiNode): boolean =>
  isSsoNode(node) &&
  node.type === "input" &&
  (node.attributes as UiNodeInputAttributes).name === SSO_UNLINK_FIELD;

export const getSsoLinkNodeId = (connectionId: string): string =>
  `${SSO_LINK_NODE_PREFIX}${connectionId}`;

export function isUiNodeBackButton(meta: UiNodeMeta) {
  return meta.label?.type === "info" && meta.label?.text === "Back";
}
