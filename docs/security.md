# Security Overview in Identity Platform Login UI

The Identity Platform Login UI is a middleware component which routes calls between the different services of the Identity Platform and serves login, settings and error pages.
This document provides an overview of security concerns related to the Identity Platform Login UI, highlighting common risks and outlining best practices to mitigate them.

What is not included in this document and regarded as out of scope:
- Upstream workloads code (refer to the workloads’ security documentation - [Ory Hydra](https://www.ory.sh/docs/hydra/security-architecture) and [Ory Kratos](https://www.ory.sh/docs/kratos/concepts/security)).

## Common risks

Similar to other web-based services, the Identity Platform Login UI faces the following typical potential security risk challenges.

**Authentication and Authorization Vulnerabilities**: Proper handling of authentication tokens and session management.

**Injection Attacks**: Inputs provided by users, especially those directly interacting with backend services, must be sanitised to avoid injection vulnerabilities such as SQL injection or command injection.

**Sensitive Data Exposure**: Personally identifiable information (PII), access tokens, and passwords should be handled with encryption and strong access controls to avoid leaks.

**Cross-Site Scripting (XSS) and Cross-Site Request Forgery (CSRF)**: Frontend security is a concern to avoid XSS, CSRF, and other web-related vulnerabilities.

**Data Breaches**: Compromised credentials or poorly protected data can lead to breaches of sensitive user information.

## Built-in Protection Mechanisms

The Identity Platform Login UI offers several built-in security features to protect from potential security issues.

### Authentication mechanisms

The Identity Platform Login UI supports multiple secure authentication methods, including:

- Multi-Factor Authentication (MFA): Adds an additional layer of security by requiring users to verify their identity through multiple steps.
- WebAuthn Passwordless Authentication: allows to authenticate users using public key cryptography. See more information [here](https://www.ory.sh/docs/kratos/passwordless/passwordless).

### Identity management

The service relies on Ory Kratos for identity management to ensure that sensitive user data is managed through industry-standard protocols and best practices.

### Password policies

Enforcing strong password policies, including minimum password length, is crucial to reduce the risk of brute force attacks. Identity Platform Login UI comes with a built-in password policy. The password is required to be at least 8 characters long, including lowercase and uppercase letters and numbers.

### Encrypted Communication

All communications between the UI and all backend services are expected to use SSL/TLS encryption to protect against man-in-the-middle attacks.

### Web Application Security

The service implements comprehensive web application security measures by adhering to industry standards. This includes input validation and sanitization, secure cookie management, and browser-based defences, such as using CSRF protection. These measures are designed to mitigate common web vulnerabilities including cross-site scripting (XSS) and other attack vectors that target web applications.

## Cryptography

### Packages used by Identity Platform Login UI

- `crypto-js` is one of the dependencies required by the react-pdf package used by the frontend service.

## Best Practices

The service comes with multiple security features by default, but implementing best practices can further safeguard against new threats. Here are a few important suggestions.

### Keep Service Updated

Regularly update the service and its dependencies to address known security vulnerabilities. Canonical provides frequent security patches and updates to ensure systems remain secure.

### Enable Data Encryption

Use encryption to protect data both during transmission (via HTTPS/TLS) and while it is stored, ensuring sensitive information remains secure.

### Set Up Monitoring and Alerts

Constantly monitor service logs for unusual activities, and configure alerts for critical events like failed login attempts, privilege escalation, or configuration changes to quickly address potential threats. The Identity Platform Login UI Charmed Operator offers integration with Canonical Observability Stack to ease that process.
