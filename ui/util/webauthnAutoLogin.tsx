const WEBAUTHN_AUTOLOGIN_KEY = "webauthn_autologin";
const WEBAUTHN_AUTOLOGIN_VALUE = "true";

export const isWebauthnAutologin = (): boolean => {
  return (
    localStorage.getItem(WEBAUTHN_AUTOLOGIN_KEY) === WEBAUTHN_AUTOLOGIN_VALUE
  );
};

export const toggleWebauthnSkip = () => {
  if (isWebauthnAutologin()) {
    localStorage.removeItem(WEBAUTHN_AUTOLOGIN_KEY);
  } else {
    localStorage.setItem(WEBAUTHN_AUTOLOGIN_KEY, WEBAUTHN_AUTOLOGIN_VALUE);
  }
};
