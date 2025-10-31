import React, { FC, useCallback, useMemo, useState } from "react";
import { useRouter } from "next/router";
import { Button, Icon } from "@canonical/react-components";
import classnames from "classnames";
import { FeatureEnabled } from "../util/featureFlags";

interface Props {
  user?: string;
}

const SelfServeNavigation: FC<Props> = ({ user }) => {
  const [menuCollapsed, setMenuCollapsed] = useState(false);

  const router = useRouter();

  const logo = useMemo(
    () => (
      <a className="p-panel__title self-serve-logo" href="./manage_details">
        <img
          className="p-panel__logo-icon image"
          alt="Canonical"
          src="https://assets.ubuntu.com/v1/b8337572-COF-tag.png"
        />
        <div className="p-heading--4 text">Canonical SSO</div>
      </a>
    ),
    [],
  );

  const navItem = (path: string, label: string) => {
    const currentPath = router.asPath;
    const pathMatcher = path.replace("manage", "");
    const ariaCurrent = currentPath.includes(pathMatcher) ? "page" : undefined;

    return (
      <li className="p-side-navigation__item">
        <a
          className="p-side-navigation__link"
          href={`./${path}`}
          aria-current={ariaCurrent}
        >
          <span className="p-side-navigation__label">{label}</span>
        </a>
      </li>
    );
  };

  const hardToggleMenu = useCallback(() => {
    setMenuCollapsed(!menuCollapsed);
  }, [setMenuCollapsed, menuCollapsed]);

  return (
    <>
      <header className="l-navigation-bar">
        <div className="p-panel is-light">
          <div className="p-panel__header">
            {logo}
            <div className="p-panel__controls">
              <Button
                dense
                className="p-panel__toggle"
                onClick={hardToggleMenu}
              >
                Menu
              </Button>
            </div>
          </div>
        </div>
      </header>
      <nav
        aria-label="main navigation"
        className={classnames("l-navigation", {
          "is-collapsed": menuCollapsed,
          "is-pinned": !menuCollapsed,
        })}
      >
        <div className="l-navigation__drawer">
          <div className="p-panel is-paper">
            <div className="p-panel__header">
              {logo}
              <div className="p-panel__controls u-hide--large u-hide--medium">
                <Button
                  appearance="base"
                  hasIcon
                  className="u-no-margin"
                  aria-label="close navigation"
                  onClick={hardToggleMenu}
                >
                  <Icon name="close" />
                </Button>
              </div>
            </div>
            <div className="p-panel__content">
              <nav className="p-side-navigation--icons" aria-label="Main">
                <ul className="p-side-navigation__list">
                  {navItem("manage_details", "Personal details")}

                  <FeatureEnabled flags={"password"}>
                    {navItem("manage_password", "Password")}
                  </FeatureEnabled>

                  <FeatureEnabled flags={"webauthn"}>
                    {navItem("manage_passkey", "Security key")}
                  </FeatureEnabled>

                  <FeatureEnabled flags={"backup_codes"}>
                    {navItem("manage_backup_codes", "Backup codes")}
                  </FeatureEnabled>

                  <FeatureEnabled flags={"totp"}>
                    {navItem("manage_secure", "Authenticator")}
                  </FeatureEnabled>

                  <FeatureEnabled flags={"account_linking"}>
                    {navItem("manage_connected_accounts", "Connected accounts")}
                  </FeatureEnabled>
                </ul>
                {user && (
                  <ul className="p-side-navigation__list self-serve-user">
                    <li className="p-side-navigation__item">
                      <div className="p-side-navigation__link">
                        <Icon
                          name="profile"
                          className="p-side-navigation__icon"
                        />
                        <span
                          className="p-side-navigation__label u-truncate"
                          title={user}
                        >
                          {user}
                        </span>
                      </div>
                    </li>
                  </ul>
                )}
              </nav>
            </div>
          </div>
        </div>
      </nav>
    </>
  );
};

export default SelfServeNavigation;
