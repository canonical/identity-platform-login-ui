# Spec Delta

## Purpose
The service-health capability checks Kratos and Hydra readiness in the background and reports dependency availability. It defines behavior when a dependency URL is not configured.

## ADDED Requirements

### Requirement: Background dependency readiness checks
The status service SHALL check Kratos and Hydra readiness periodically and SHALL track the latest result for each dependency. The status service SHALL expose the latest result for each dependency to the status endpoint.

#### Scenario: Configured dependency reports readiness
- **WHEN** a dependency URL is configured
- **THEN** the status service SHALL query that dependency's readiness endpoint periodically and SHALL track the latest result

#### Scenario: Failed check reports unavailable
- **WHEN** a readiness query to a configured dependency fails
- **THEN** the status service SHALL report that dependency as unavailable

### Requirement: Unconfigured dependency reports unavailable without network calls
When a dependency URL is empty, the status service SHALL report that dependency as unavailable. The status service MUST NOT attempt a network call for that dependency.

#### Scenario: Empty Kratos URL
- **WHEN** `KRATOS_PUBLIC_URL` is unset or empty
- **THEN** the status service SHALL report Kratos as unavailable and MUST NOT call the Kratos SDK

#### Scenario: Empty Hydra URL
- **WHEN** `HYDRA_ADMIN_URL` is unset or empty
- **THEN** the status service SHALL report Hydra as unavailable and MUST NOT call the Hydra SDK

#### Scenario: No error logs for unconfigured dependencies
- **WHEN** a dependency URL is unset or empty
- **THEN** the service MUST NOT emit error log entries from that dependency's readiness check

### Requirement: Startup warning for unconfigured dependencies
When a dependency URL is empty, the service SHALL log one warning for that dependency at startup.

#### Scenario: Warning logged once
- **WHEN** the service starts with a dependency URL unset or empty
- **THEN** the service SHALL log exactly one warning naming the unconfigured dependency

### Requirement: Dependency URLs come from configuration
The web layer SHALL pass the configured Kratos public URL and Hydra admin URL to the status service. Router options SHALL provide both URLs.

#### Scenario: Router passes both URLs
- **WHEN** the application builds its router
- **THEN** the router SHALL supply the configured Kratos public URL and Hydra admin URL to the status service
