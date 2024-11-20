// see https://www.ory.sh/docs/kratos/concepts/ui-messages
import { UiNode } from "@ory/client";

export const ORY_LABEL_ID_ADD_SECURITY_KEY = 1050012;
export const ORY_LABEL_ID_NAME_OF_THE_SECURITY_KEY = 1050013;
export const ORY_LABEL_ID_REMOVE_SECURITY_ID = 1050018;
const ORY_LABEL_BACKUP_CODE_CREATE = 1050008;
const ORY_LABEL_BACKUP_CODE_CONFIRM_TEXT = 1050010;
const ORY_LABEL_BACKUP_CODE_CONFIRM = 1050011;
const ORY_LABEL_BACKUP_CODE_VIEW = 1050007;
const ORY_LABEL_BACKUP_CODE_DEACTIVATE = 1050016;
const ORY_LABEL_USE_AUTHENTICATOR = 1010009;
const ORY_LABEL_USE_BACKUP_CODE = 1010010;

type NodeWithLabel = UiNode & { meta: { label: object } };

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