import {
  LoginFlow,
  UiNode,
  UiNodeInputAttributes,
  UpdateLoginFlowBody,
} from "@ory/client";
import { CheckboxInput, Spinner } from "@canonical/react-components";
import { AxiosError } from "axios";
import type { NextPage } from "next";
import { useRouter } from "next/router";
import { useEffect, useState, useCallback, useRef } from "react";
import React from "react";
import {
  getInPlaceErrorMessage,
  handleFlowError,
  isKratosError,
} from "../util/handleFlowError";
import { Flow } from "../components/Flow";
import { FlowMessages } from "../components/FlowMessages";
import { RedirectingNotice } from "../components/RedirectingNotice";
import { getRedirectLabel, useLabelledRedirect } from "../util/redirectTo";
import {
  IdentifierFirstError,
  kratos,
  loginIdentifierFirst,
} from "../api/kratos";
import { FlowResponse } from "./consent";
import PageLayout from "../components/PageLayout";
import { replaceAuthLabel } from "../util/replaceAuthLabel";
import {
  UpdateLoginFlowWithLookupSecretMethod,
  UpdateLoginFlowWithOidcMethod,
  UpdateLoginFlowWithPasswordMethod,
} from "@ory/client/api";
import {
  isErrorAnsweredByBackend,
  isSignInEmailInput,
  isSignInWithHardwareKey,
  isSignInWithPassword,
  isSsoNode,
  isTenantChoice,
  isTenantNode,
} from "../util/constants";
import {
  isWebauthnAutologin,
  toggleWebauthnSkip,
} from "../util/webauthnAutoLogin";
import { getCsrfNode, getCsrfToken } from "../util/getCsrfNode";

type AppConfig = {
  oidc_webauthn_sequencing_enabled?: boolean;
};

const getTitleSuffix = (reqName: string, reqDomain: string) => {
  if (reqName && reqDomain) {
    return ` to ${reqName} on ${reqDomain}`;
  }
  if (reqName) {
    return ` to ${reqName}`;
  }
  if (reqDomain) {
    return ` to ${reqDomain}`;
  }
  return "";
};

const resolveLoginTitle = (
  isIdentifierFirst: boolean,
  isAuthCode: UiNode | undefined,
  titleSuffix: string,
) => {
  if (isIdentifierFirst) {
    return "Sign in";
  }

  if (isAuthCode) {
    return "Verify your identity";
  }

  return `Sign in${titleSuffix}`;
};

const Login: NextPage = () => {
  const [flow, setFlow] = useState<LoginFlow>();
  const [isSequencedLogin, setSequencedLogin] = useState(false);
  // An error the backend answered with instead of a flow, shown in place.
  const [inPlaceError, setInPlaceError] = useState<string>();
  // A tenant or company sign-in pick in flight, and why the last one failed.
  const [pickPending, setPickPending] = useState(false);
  const pickInFlight = useRef(false);
  const [pickError, setPickError] = useState<string>();
  // Set while the browser is being sent to a company sign-in.
  const [redirectLabel, redirectWithLabel] = useLabelledRedirect();
  const isAuthCode = flow?.ui.nodes.find((node) => node.group === "totp");
  // Only auto-select the WebAuthn form when WebAuthn is the *sole* 2FA method.
  // When TOTP is also registered the selection page must show both options so the
  // user can choose either; forcing WebAuthn directly would hide TOTP and, when
  // the ceremony fails, clicking "I want to use another method" restarts the
  // entire login at the identifier page (see issue #839).
  const is2FaWebauthn =
    flow?.requested_aal === "aal2" &&
    !isAuthCode &&
    flow?.ui.nodes.find((node) => node.group === "webauthn") !== undefined;

  const isIdentifierFirst =
    flow?.ui.nodes.some(
      (node) =>
        node.attributes.node_type === "input" &&
        (node.attributes as UiNodeInputAttributes).name === "method" &&
        (node.attributes as UiNodeInputAttributes).value === "identifier_first",
    ) ?? false;

  useEffect(() => {
    void fetch("../api/v0/app-config")
      .then((response) => {
        return response.json() as Promise<AppConfig>;
      })
      .then((data) => {
        setSequencedLogin(data.oidc_webauthn_sequencing_enabled ?? false);
      })
      .catch(console.error);
  }, []);

  // Get ?flow=... from the URL
  const router = useRouter();
  const {
    return_to: returnTo,
    flow: flowId,
    // Refresh means we want to refresh the session. This is needed, for example, when we want to update the password
    // of a user.
    refresh,
    // AAL = Authorization Assurance Level. This implies that we want to upgrade the AAL, meaning that we want
    // to perform two-factor authentication/verification.
    aal,
    login_challenge,
    use_backup_code: useBackupCode,
    email,
    invalid_method,
    pw_changed: pwChanged,
  } = router.query;

  const redirectToErrorPage = () => {
    const idParam = flowId ? `?id=${flowId.toString()}` : "";
    window.location.href = `./error${idParam}`;
  };

  // Keep the user on the login page for an error that is shown in place,
  // pass any other error on.
  const handleInPlaceError = (err: Error) => {
    const data =
      err instanceof IdentifierFirstError
        ? err.data
        : (err as AxiosError).response?.data;
    const message = getInPlaceErrorMessage(data);
    if (!message) {
      return Promise.reject(err);
    }
    setInPlaceError(message);
  };

  useEffect(() => {
    // If the router is not ready yet, do nothing.
    if (!router.isReady) {
      return;
    }

    if (flowId && flow) {
      return;
    }

    // If ?flow=.. was in the URL, we fetch it
    if (flowId) {
      kratos
        .getLoginFlow({ id: String(flowId) })
        .then((res) => setFlow(res.data))
        .catch(handleFlowError("login", setFlow))
        .catch(handleInPlaceError)
        .catch(redirectToErrorPage);
      return;
    }

    const getReturnTo = () => {
      if (returnTo) {
        return String(returnTo);
      }
      if (login_challenge) {
        return undefined;
      }
      return window.location.pathname;
    };

    // Otherwise we initialize it
    kratos
      .createBrowserLoginFlow({
        refresh: Boolean(refresh),
        aal: aal ? String(aal) : undefined,
        returnTo: getReturnTo(),
        loginChallenge: login_challenge ? String(login_challenge) : undefined,
      })
      .then(async ({ data }: FlowResponse) => {
        if (data.redirect_to !== undefined) {
          const label = getRedirectLabel(data);
          if (label) {
            redirectWithLabel(data.redirect_to, label);
            return;
          }
          const addendum = data.redirect_to.includes("?") ? "&" : "?";
          const pwParam = pwChanged
            ? `${addendum}pw_changed=${pwChanged as string}`
            : "";
          window.location.href = `${data.redirect_to}${pwParam}`;
          return;
        }

        setFlow(data);

        await router.replace(
          {
            query: {
              ...router.query,
              flow: data.id,
            },
          },
          undefined,
          { shallow: true },
        );
      })
      .catch(handleFlowError("login", setFlow))
      .catch(handleInPlaceError)
      .catch(redirectToErrorPage);
  }, [
    flowId,
    router,
    router.isReady,
    aal,
    refresh,
    returnTo,
    flow,
    login_challenge,
  ]);

  const handleSubmit = useCallback(
    (values: UpdateLoginFlowBody) => {
      const getMethod = () => {
        if (values.method === "identifier_first") {
          return "identifier_first";
        }
        if ((values as UpdateLoginFlowWithOidcMethod).provider) {
          return "oidc";
        }
        if (values.method === "webauthn") {
          return "webauthn";
        }
        // A backup code submitted with Enter carries no method: the code
        // itself says which one it is.
        if (
          values.method === "lookup_secret" ||
          (values as UpdateLoginFlowWithLookupSecretMethod).lookup_secret
        ) {
          return "lookup_secret";
        }
        if (isAuthCode) {
          return "totp";
        }
        return "password";
      };
      const method = getMethod();

      const isPasswordMissing = !(values as UpdateLoginFlowWithPasswordMethod)
        .password;

      const setEmptyPassword = () => {
        (values as UpdateLoginFlowWithPasswordMethod).password = "";
      };

      if (method === "password" && isPasswordMissing) {
        setEmptyPassword();
      }

      if (method === "identifier_first") {
        const flowId = String(flow?.id);

        return loginIdentifierFirst(
          flowId,
          values,
          method,
          flow,
          typeof login_challenge === "string"
            ? login_challenge
            : flow?.oauth2_login_challenge,
        )
          .then((data) => {
            if ("redirect_to" in data) {
              redirectWithLabel(data.redirect_to, getRedirectLabel(data));
            } else {
              setFlow(data);
            }
          })
          .catch(handleInPlaceError)
          .catch(redirectToErrorPage);
      }

      return kratos
        .updateLoginFlow({
          flow: String(flow?.id),
          updateLoginFlowBody: {
            ...values,
            method,
          } as UpdateLoginFlowBody,
        })
        .then(({ data }) => {
          if ("state" in data && data.state === "choose_method") {
            setFlow(data as unknown as LoginFlow);
            return;
          }
          if ("redirect_to" in data) {
            redirectWithLabel(
              data.redirect_to as string,
              getRedirectLabel(data),
            );
            return;
          }
          if (flow?.return_to) {
            window.location.href = flow.return_to;
            return;
          }
        })
        .catch(handleFlowError("login", setFlow))
        .catch((err: AxiosError<LoginFlow>) => {
          if (err.response?.status === 400) {
            if (
              typeof err.response.data === "object" &&
              "ui" in err.response.data
            ) {
              setFlow(err.response.data);
              return;
            }

            // A Kratos error handleFlowError does not know how to recover from
            redirectToErrorPage();
            return;
          }

          if (getInPlaceErrorMessage(err.response?.data)) {
            return handleInPlaceError(err);
          }

          // A Kratos error passed on with the status Kratos gave it
          if (isKratosError(err.response?.data)) {
            redirectToErrorPage();
            return;
          }

          if (
            // eslint-disable-next-line @typescript-eslint/no-base-to-string
            err.response?.data.toString().trim() ===
            "choose a different login method"
          ) {
            const url = new URL(window.location.href);
            url.searchParams.set(
              "email",
              (values as UpdateLoginFlowWithPasswordMethod).identifier,
            );
            url.searchParams.set("invalid_method", "1");
            window.location.href = url.toString();
            return;
          }

          return Promise.reject(err);
        });
    },
    [flow, router, login_challenge],
  );

  // Called where the user acts, never from handleSubmit: the page also calls
  // that while rendering.
  const clearErrors = () => {
    setInPlaceError(undefined);
    setPickError(undefined);
  };

  const reqName = flow?.oauth2_login_request?.client?.client_name ?? "";
  const reqDomain = flow?.oauth2_login_request?.client?.client_uri
    ? new URL(flow.oauth2_login_request.client.client_uri).hostname
    : "";
  const titleSuffix = getTitleSuffix(reqName, reqDomain);
  const title = resolveLoginTitle(isIdentifierFirst, isAuthCode, titleSuffix);

  const filterFlow = (flow: LoginFlow | undefined): LoginFlow => {
    if (!flow) {
      return flow as unknown as LoginFlow;
    }

    return {
      ...flow,
      ui: {
        ...flow.ui,
        nodes: flow.ui.nodes.filter(({ group }) => {
          return useBackupCode
            ? group !== "totp" && group !== "webauthn"
            : group !== "lookup_secret";
        }),
      },
    };
  };

  let isWebauthn = false;
  const supportsWebauthn = flow?.ui.nodes.some(
    (node) => node.group === "webauthn",
  );

  const renderFlow: LoginFlow | undefined = flow
    ? isIdentifierFirst
      ? {
          ...flow,
          ui: {
            ...flow.ui,
            nodes: flow.ui.nodes.filter((n: UiNode) => {
              if (
                n.attributes.node_type === "input" &&
                typeof (n.attributes as UiNodeInputAttributes).name === "string"
              ) {
                const name = (n.attributes as UiNodeInputAttributes).name;
                return (
                  name === "identifier" ||
                  name === "csrf_token" ||
                  name === "method"
                );
              }
              return false;
            }),
          },
        }
      : isAuthCode || is2FaWebauthn
        ? filterFlow(replaceAuthLabel(flow))
        : flow
    : undefined;

  if (renderFlow?.ui) {
    const urlParams = new URLSearchParams(window.location.search);
    const hasWebauthnInUrlParam = urlParams.get("webauthn") === "true";
    const hasOnlyWebauthnNodes =
      renderFlow.ui.nodes.filter(
        (node) => node.group !== "webauthn" && node.group !== "default",
      ).length === 0;

    isWebauthn =
      (hasWebauthnInUrlParam || hasOnlyWebauthnNodes || is2FaWebauthn) &&
      !invalid_method &&
      !useBackupCode;

    renderFlow.ui.nodes = renderFlow?.ui.nodes.filter((node) => {
      // show webauthn elements in dedicated step after it is selected
      if (isWebauthn) {
        return node.group === "webauthn" || node.group === "default";
      }
      // hide webauthn everywhere else
      return node.group !== "webauthn";
    });

    // add security key option that looks like an oidc input
    if (!isWebauthn && !useBackupCode && supportsWebauthn) {
      renderFlow.ui.nodes.push({
        attributes: {
          type: "url",
          node_type: "input",
          name: "",
          disabled: false,
        },
        group: "webauthn",
        type: "input",
        messages: [],
        meta: {
          label: {
            id: 1,
            text: "Sign in with Security key",
            type: "info",
          },
        },
      });
    }

    // The tenant list and the company sign-ins, which the backend adds.
    const isPick = (node: UiNode) => isTenantNode(node) || isSsoNode(node);

    // ensure oidc options are presented after username/password inputs,
    // and the tenant list after both
    renderFlow.ui.nodes.sort((a, b) => {
      const toValue = (node: UiNode) =>
        isTenantNode(node) ? 2 : node.group === "oidc" ? 1 : -1;
      return toValue(a) - toValue(b);
    });

    // A tenant or company sign-in pick posts only itself: a value kept in the
    // form from an earlier pick must not ride along with a password.
    renderFlow.ui.nodes.filter(isPick).forEach((node) => {
      const label = node.meta.label;
      if (!label || node.attributes.node_type !== "input") {
        return;
      }
      const attributes = node.attributes as UiNodeInputAttributes;
      attributes.disabled = pickPending;
      node.meta.label = {
        ...label,
        context: {
          ...label.context,
          onClick: () => {
            if (pickInFlight.current) {
              return;
            }
            pickInFlight.current = true;
            setPickPending(true);
            clearErrors();
            void handleSubmit({
              csrf_token: getCsrfToken(renderFlow.ui.nodes),
              [attributes.name]: attributes.value as string,
            } as unknown as UpdateLoginFlowBody)
              .catch(() => setPickError("Something went wrong. Try again."))
              .finally(() => {
                pickInFlight.current = false;
                setPickPending(false);
              });
          },
        },
      };
    });

    const firstTenant = renderFlow.ui.nodes.find(isTenantChoice);
    // The tenant list is the whole screen: the account's own sign-ins belong
    // to the tenant that accepts them, shown once it is picked.
    if (firstTenant?.meta.label) {
      firstTenant.meta.label = {
        ...firstTenant.meta.label,
        context: {
          ...firstTenant.meta.label?.context,
          beforeComponent: (
            <p className="p-heading--5 u-sv1">Choose a tenant to sign in to</p>
          ),
        },
      };
    }

    // autosubmit webauthn in case email is provided (1FA only — in AAL2 the
    // identity is already known from the session, so skip auto-submit)
    const email = urlParams.get("email");
    // an error the submission was answered with is shown in place instead
    if (
      isWebauthn &&
      email &&
      flow?.requested_aal !== "aal2" &&
      !inPlaceError
    ) {
      void handleSubmit({
        method: "webauthn",
        identifier: email,
        csrf_token: getCsrfToken(renderFlow?.ui.nodes),
      }).catch(() => {
        if (flow?.return_to) {
          window.location.href = flow.return_to;
        }
      });

      return null;
    }
  }

  if (redirectLabel) {
    return (
      <PageLayout title="Sign in">
        <RedirectingNotice label={redirectLabel} />
      </PageLayout>
    );
  }

  if (!flow) {
    // No flow to render, e.g. single sign-on was unavailable while creating it.
    return inPlaceError ? (
      <PageLayout title="Sign in">
        <FlowMessages
          messages={[{ id: 0, type: "error", text: inPlaceError }]}
        />
      </PageLayout>
    ) : undefined;
  }

  // Flow-level messages, e.g. a rejection by the company's identity provider.
  const flowMessages = [
    ...(flow.ui.messages ?? []),
    ...(inPlaceError
      ? [{ id: 0, type: "error" as const, text: inPlaceError }]
      : []),
    ...(pickError ? [{ id: 0, type: "error" as const, text: pickError }] : []),
  ];
  // An error FlowMessages leaves out is not one the user gets to see here.
  const hasFlowError = flowMessages.some(
    (message) => message.type === "error" && !isErrorAnsweredByBackend(message),
  );

  // When WebAuthn is shown but TOTP is also registered, "I want to use another
  // method" should return to the TOTP selection page (strip ?webauthn=true and
  // ?email= from the URL).  When WebAuthn is the sole 2FA method there is no
  // other method to fall back to, so point at flow.return_to instead.
  const anotherMethodUrl = isAuthCode
    ? (() => {
        const url = new URL(window.location.href);
        url.searchParams.delete("webauthn");
        url.searchParams.delete("email");
        return url.toString();
      })()
    : flow?.return_to;

  renderFlow?.ui.nodes.map((node) => {
    if (isSignInWithPassword(node)) {
      node.meta.label.text = "Sign in";
    }
    if (isSignInEmailInput(node)) {
      node.meta.label.text = "Email";
    }
    if (isSignInEmailInput(node) && email && invalid_method) {
      (node.attributes as unknown as { value: string }).value =
        typeof email === "string" ? email : email[email.length - 1];
      node.messages.push({
        id: 1,
        type: "error",
        text: "Invalid login method",
      });
    }
    if (isSignInWithHardwareKey(node) && isSequencedLogin) {
      node.meta.label.text = "Sign in using Security key";
      node.meta.label.context = {
        ...node.meta.label.context,
        icon: "lock-locked",
      };
    }
    return node;
  });

  // automatically forward to single oidc provider if it is the only option
  const csrfNode = getCsrfNode(renderFlow?.ui.nodes);
  const isSingleOidcOption =
    isSequencedLogin &&
    // never forward away from an error the user has not seen yet
    !hasFlowError &&
    renderFlow?.ui.nodes.length === 2 &&
    renderFlow?.ui.nodes[1].group === "oidc" &&
    csrfNode !== undefined;
  if (isSingleOidcOption) {
    const oidcNode = renderFlow?.ui.nodes[1];
    const oidcAttributes = oidcNode.attributes as UiNodeInputAttributes;
    const csrfAttributes = csrfNode.attributes as UiNodeInputAttributes;
    void handleSubmit({
      method: "oidc",
      provider: oidcAttributes.value as string,
      csrf_token: csrfAttributes.value as string,
    });
  }

  return (
    <PageLayout title={title}>
      <FlowMessages messages={flowMessages} />
      {isSingleOidcOption ? (
        <p className="u-text--muted">
          <Spinner style={{ marginRight: "0.5rem" }} />
          You will be redirected to the login provider.
        </p>
      ) : (
        <>
          {isWebauthn && isSequencedLogin && (
            <p className="u-text--muted">
              Additional authentication needed to get access{" "}
              {getTitleSuffix(reqName, reqDomain)}
            </p>
          )}
          {flow ? (
            <Flow
              onSubmit={(values: UpdateLoginFlowBody) => {
                clearErrors();
                return handleSubmit(values);
              }}
              flow={renderFlow}
            />
          ) : (
            <Spinner />
          )}
          {isWebauthn && !isSequencedLogin && (
            <a href={anotherMethodUrl}>I want to use another method</a>
          )}
          {isWebauthn && isSequencedLogin && (
            <CheckboxInput
              label="Don't show again"
              defaultChecked={isWebauthnAutologin()}
              onChange={toggleWebauthnSkip}
            />
          )}
        </>
      )}
    </PageLayout>
  );
};

export default Login;
