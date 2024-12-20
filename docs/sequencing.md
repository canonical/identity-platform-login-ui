## OIDC-WebAuthn Sequencing

This application exposes an `OIDC_WEBAUTHN_SEQUENCING_ENABLED` environment variable, which defaults to `false`.

**Do not enable this option unless you are sure that this feature applies to your deployment.**

If set to `true`, Login UI will enforce setting up a WebAuthn key (e.g. with YubiKey or Google Password Manager on Android) right after signing in with an external identity provider (Google, GitHub, Entra ID, ...), given that the user has not done it previously.
Users will be asked to complete it as the second authentication factor on every login flow, regardless of the authentication controls enforced by the identity provider.

Enabling this option works on the assumption that your Kratos config has the `passwordless` flag set to `false`:

```yaml
selfservice:
  methods:
    webauthn:
      enabled: True
      config:
        passwordless: false
```

This is currently not supported in the Kratos Charmed Operator.

### Testing

In order to test this feature:

1. Switch the aforementioned flag to `false` and disable the `totp` and `password` methods in the [kratos](https://github.com/canonical/identity-platform-login-ui/blob/main/docker/kratos/kratos.yml) configuration for Docker.

2. Set environment variables:

    ```bash
    export OIDC_WEBAUTHN_SEQUENCING_ENABLED="true"
    ```

3. Follow the [instructions](https://github.com/canonical/identity-platform-login-ui/blob/main/README.md#try-it-out) to integrate an external identity provider and run the docker setup.

4. Create a hydra client for an application you'll use to test the login flow. Save the client id and secret.

    ```docker
    docker exec <hydra-container> \
    hydra create client \
        --endpoint http://127.0.0.1:4445 \
        --name <app-name> \
        --grant-type authorization_code,refresh_token \
        --response-type code,id_token \
        --format json \
        --scope openid,offline_access,email \
        --redirect-uri <app-redirect-uri>
    ```

5. Deploy an application that supports OIDC. We'll use Grafana as an example:

    ```docker
    docker run -d --name=grafana -p 2345:2345 --network identity-platform-login-ui_intranet \
    -e "GF_SERVER_HTTP_PORT=2345" \
    -e "GF_AUTH_GENERIC_OAUTH_ENABLED=true" \
    -e "GF_AUTH_GENERIC_OAUTH_AUTH_ALLOWED_DOMAINS=hydra,localhost" \
    -e "GF_AUTH_GENERIC_OAUTH_NAME=Identity Platform" \
    -e "GF_AUTH_GENERIC_OAUTH_CLIENT_ID=<client-id>" \
    -e "GF_AUTH_GENERIC_OAUTH_CLIENT_SECRET=<client-secret>" \
    -e "GF_AUTH_GENERIC_OAUTH_SCOPES=openid offline_access email" \
    -e "GF_AUTH_GENERIC_OAUTH_AUTH_URL=http://localhost:4444/oauth2/auth" \
    -e "GF_AUTH_GENERIC_OAUTH_TOKEN_URL=http://hydra:4444/oauth2/token" \
    -e "GF_AUTH_GENERIC_OAUTH_API_URL=http://hydra:4444/userinfo" \
    grafana/grafana
    ```

6. Go to the application's login page and click on `Sign in with Identity Platform`. Then, sign in with the integrated identity provider. Upon a successful authentication, you will be asked to create a WebAuthn key.
