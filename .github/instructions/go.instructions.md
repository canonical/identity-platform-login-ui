# Go Coding Conventions - Complete Reference

This document provides comprehensive Go coding standards for the identity-platform-login-ui project.

## Error Handling

### Error Messages

**Format:**
- Start with lowercase verbs: "cannot", "failed to", "invalid"
- Be concise - avoid verbose explanations
- Example: `fmt.Errorf("cannot fetch flow: %w", err)` ✅
- Not: `fmt.Errorf("An error occurred while attempting to fetch the flow")` ❌

### When to Add Context (`%w` vs `%v`)

**Use `%w` (wrap) when:**
- Caller needs to inspect/recover from the specific error
- This makes the inner error part of your public API
- Example: Custom domain errors that handlers need to map to HTTP codes

**Use `%v` (paste) when:**
- Error is unrecoverable or implementation detail
- Prevents caller dependency on implementation
- Most library/service code falls into this category

**Examples:**
```go
// Good - adds context at package boundary
if err := s.kratos.GetLoginFlow(ctx, flowID, cookies); err != nil {
    return nil, fmt.Errorf("failed to fetch login flow: %v", err)
}

// Bad - adds noise without value
if err := helper(); err != nil {
    return fmt.Errorf("helper failed: %v", err)
}

// Good - wrapping for caller to inspect
if errors.Is(err, kratos.ErrNotFound) {
    return fmt.Errorf("flow not found: %w", err)
}
```

### Error Return Values

**Rules:**
- When error is `nil`: return valid data
- When error is non-`nil`: return zero values
- Never return partial data with an error

```go
// Good
func (s *Service) GetFlow(ctx context.Context, id string) (*Flow, []*http.Cookie, error) {
    flow, cookies, err := s.kratos.GetLoginFlow(ctx, id, nil)
    if err != nil {
        return nil, nil, err  // Zero values for data when error present
    }
    return flow, cookies, nil
}

// Bad
func (s *Service) GetFlow(ctx context.Context, id string) (*Flow, []*http.Cookie, error) {
    flow, cookies, err := s.kratos.GetLoginFlow(ctx, id, nil)
    return flow, cookies, err  // Returns partial data even if err != nil
}
```

### Logging Errors

**Where to log:**
- Log errors at the **handler layer** (`pkg/*/handlers.go`)
- Do NOT log in service or client layers
- Prevents duplicate log entries

**Exception:**
- Debug-level logging is acceptable for troubleshooting

```go
// Good - handler logs the error
func (a *API) handleLogin(w http.ResponseWriter, r *http.Request) {
    flow, err := a.service.GetLoginFlow(r.Context(), flowID)
    if err != nil {
        a.logger.Errorf("failed to get login flow: %s", err)  // Log here
        w.WriteHeader(http.StatusInternalServerError)
        return
    }
    // ...
}

// Service layer doesn't log
func (s *Service) GetLoginFlow(ctx context.Context, flowID string) (*Flow, error) {
    flow, err := s.kratos.GetFlow(ctx, flowID)
    if err != nil {
        return nil, err  // No logging, just return
    }
    return flow, nil
}
```

## Naming Conventions

### Receivers

**Always name receivers**, even if unused:
```go
// Good
func (s *Service) Method() {}

// Bad
func (*Service) Method() {}
```

**Use consistent receiver names** across all methods of a type:
- `a *API`
- `s *Service`
- `c *Client`

### Functions

**Doc Comments:**
- All functions (even unexported ones used across files) must have doc comments
- Format: Start with function name
- Keep concise (1-2 sentences)
- Describe purpose, not implementation

```go
// Good
// GetLoginFlow retrieves the login flow for the given ID.
func (s *Service) GetLoginFlow(ctx context.Context, flowID string) (*Flow, error) {
    // ...
}

// Bad - no doc comment
func (s *Service) GetLoginFlow(ctx context.Context, flowID string) (*Flow, error) {
    // ...
}

// Bad - describes implementation
// GetLoginFlow calls the Kratos client FrontendAPI GetLoginFlow method
// and then wraps the response in our internal Flow struct after validation.
func (s *Service) GetLoginFlow(ctx context.Context, flowID string) (*Flow, error) {
    // ...
}
```

### Variables

**Concise names without type suffixes:**
```go
// Good
flow := getFlow()
flowID := "abc123"

// Bad
flowObject := getFlow()
flowString := "abc123"
```

**Exception:** When disambiguating different types of the same concept:
```go
flowID string
flowData *Flow
```

**Use US spelling throughout.**

## Code Structure

### Avoid Pyramids of Doom

Use early returns to keep code flat:

```go
// Good - flat structure
func (s *Service) processFlow(ctx context.Context, flowID string) error {
    if err := s.validateFlow(flowID); err != nil {
        return err
    }
    if err := s.executeFlow(ctx, flowID); err != nil {
        return err
    }
    return nil
}

// Bad - nested indentation
func (s *Service) processFlow(ctx context.Context, flowID string) error {
    if err := s.validateFlow(flowID); err == nil {
        if err := s.executeFlow(ctx, flowID); err == nil {
            return nil
        } else {
            return err
        }
    } else {
        return err
    }
}
```

### Variable Declaration

**Prefer `:=` for most declarations:**
```go
// Good
flow := getFlow()
count := 0

// Bad
var flow = getFlow()
```

**Use `var` only when zero value is intentionally assigned before any reads:**
```go
// Good - zero value is used
var count int
for _, item := range items {
    count += item.Value
}

// Bad - value assigned immediately
var flow = getFlow()  // Should use flow := getFlow()
```

**Never use `var` with explicit initialization** (verbose, no advantage):
```go
// Bad
var flow Flow = getFlow()

// Good
flow := getFlow()
```

## Interface Conventions

### Interface Declarations

**Always include parameter names for clarity:**

```go
// Good
type ServiceInterface interface {
    GetLoginFlow(ctx context.Context, flowID string, cookies []*http.Cookie) (*LoginFlow, []*http.Cookie, error)
}

// Bad
type ServiceInterface interface {
    GetLoginFlow(context.Context, string, []*http.Cookie) (*LoginFlow, []*http.Cookie, error)
}
```

### Struct Initialization

**Always specify field names**, never use anonymous initialization:

```go
// Good
return &API{
    service: svc,
    logger:  logger,
}

// Bad - fragile to field reordering
return &API{svc, logger}
```

## Function Conventions

### Return Values

**Prefer returning `*Struct` over `Struct` (value):**
```go
// Good
func NewService() *Service {
    return &Service{}
}

// Acceptable only for small, immutable data
func GetCount() int {
    return 42
}
```

**Multiple return values are common:**
```go
func (s *Service) GetFlow(ctx context.Context, id string) (*Flow, []*http.Cookie, error)
func (s *Service) UpdateFlow(ctx context.Context, id string) (*Redirect, *Success, []*http.Cookie, error)
```

**Never use bare returns** even with named return values:
```go
// Bad
func GetFlow() (flow *Flow, err error) {
    flow = &Flow{}
    return  // Bare return
}

// Good
func GetFlow() (flow *Flow, err error) {
    flow = &Flow{}
    return flow, nil
}
```

### Passing Structs

**Always pass and receive pointers to structs:**
```go
// Good
func Process(cfg *Config) error

// Bad
func Process(cfg Config) error
```

## Panic Usage

Panics are acceptable **only** when:
1. The fault is on the caller (API misuse)
2. Code is used where error handling isn't possible (e.g., `init()` functions)

```go
// Acceptable - caller misuse
func NewService(logger LoggerInterface) *Service {
    if logger == nil {
        panic("logger is required")
    }
    return &Service{logger: logger}
}

// Not acceptable - use error return
func GetFlow(id string) *Flow {
    if id == "" {
        panic("id is required")  // Should return error instead
    }
    // ...
}
```

## Testing Standards

### Table-Driven Tests

**Always use table-driven tests:**

```go
func TestGetLoginFlow(t *testing.T) {
    tests := []struct{
        name           string
        flowID         string
        mockSetup      func(*gomock.Controller) KratosClientInterface
        expectedFlow   *Flow
        expectedError  error
    }{{
        name:   "success",
        flowID: "flow123",
        mockSetup: func(ctrl *gomock.Controller) KratosClientInterface {
            mock := NewMockKratosClientInterface(ctrl)
            mock.EXPECT().GetLoginFlow(gomock.Any(), "flow123", gomock.Any()).
                Return(&Flow{ID: "flow123"}, nil, nil)
            return mock
        },
        expectedFlow: &Flow{ID: "flow123"},
        expectedError: nil,
    }, {
        name:   "flow not found",
        flowID: "missing",
        mockSetup: func(ctrl *gomock.Controller) KratosClientInterface {
            mock := NewMockKratosClientInterface(ctrl)
            mock.EXPECT().GetLoginFlow(gomock.Any(), "missing", gomock.Any()).
                Return(nil, nil, kratos.ErrNotFound)
            return mock
        },
        expectedFlow: nil,
        expectedError: kratos.ErrNotFound,
    }}

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctrl := gomock.NewController(t)
            defer ctrl.Finish()

            kratosClient := tt.mockSetup(ctrl)
            service := NewService(kratosClient, tracer, monitor, logger)

            flow, err := service.GetLoginFlow(context.Background(), tt.flowID)

            if err != tt.expectedError {
                t.Errorf("expected error %v, got %v", tt.expectedError, err)
            }
            // Compare flow...
        })
    }
}
```

### Assertions

**Use only standard library `testing` package** - NO testify or external assertion libraries:

```go
// Good
if got != want {
    t.Errorf("expected %v, got %v", want, got)
}

// Bad - don't use testify even though legacy code has it
assert.Equal(t, want, got)
require.NoError(t, err)
```

### Mock Expectations

**Always expect tracer.Start() calls:**
```go
mockTracer.EXPECT().Start(ctx, "kratos.Service.GetLoginFlow").
    Times(1).
    Return(ctx, trace.SpanFromContext(ctx))
```

**Format:** `"package.Type.Method"` (e.g., `"kratos.Service.GetLoginFlow"`)

## File Headers

**Every Go file must start with:**
```go
// Copyright 2025 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0

package mypackage
```
