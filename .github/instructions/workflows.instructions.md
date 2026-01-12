# Common Development Patterns - Step-by-Step Guides

This document provides detailed, step-by-step instructions for common development tasks.

## Adding a New API Handler (Backend)

### Step 1: Create Interface Definitions

Create `pkg/newfeature/interfaces.go`:

```go
// Copyright 2025 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0

package newfeature

import (
    "context"
    "net/http"
)

type KratosClientInterface interface {
    // Define methods you need from Kratos
    FrontendApi() kClient.FrontendAPI
}

type ServiceInterface interface {
    DoSomething(ctx context.Context, param string) (*Result, error)
    DoSomethingElse(ctx context.Context, data *Data) error
}
```

### Step 2: Implement Service Layer

Create `pkg/newfeature/service.go`:

```go
// Copyright 2025 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0

package newfeature

import (
    "context"
    
    "github.com/canonical/identity-platform-login-ui/internal/logging"
    "github.com/canonical/identity-platform-login-ui/internal/monitoring"
    "github.com/canonical/identity-platform-login-ui/internal/tracing"
)

type Service struct {
    kratos  KratosClientInterface
    
    tracer  tracing.TracingInterface
    monitor monitoring.MonitorInterface
    logger  logging.LoggerInterface
}

// DoSomething performs the main business logic.
func (s *Service) DoSomething(ctx context.Context, param string) (*Result, error) {
    ctx, span := s.tracer.Start(ctx, "newfeature.Service.DoSomething")
    defer span.End()
    
    // Business logic here
    result := &Result{Value: param}
    return result, nil
}

// NewService creates a new newfeature service.
func NewService(
    kratos KratosClientInterface,
    tracer tracing.TracingInterface,
    monitor monitoring.MonitorInterface,
    logger logging.LoggerInterface,
) *Service {
    return &Service{
        kratos:  kratos,
        tracer:  tracer,
        monitor: monitor,
        logger:  logger,
    }
}
```

**Note:** tracer, monitor, logger are **always last** in the parameter list.

### Step 3: Create API Handler

Create `pkg/newfeature/handlers.go`:

```go
// Copyright 2025 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0

package newfeature

import (
    "encoding/json"
    "net/http"
    
    "github.com/go-chi/chi/v5"
    
    "github.com/canonical/identity-platform-login-ui/internal/logging"
)

type API struct {
    service ServiceInterface
    logger  logging.LoggerInterface
}

// RegisterEndpoints registers all API endpoints with the router.
func (a *API) RegisterEndpoints(mux *chi.Mux) {
    mux.Get("/api/newfeature", a.handleGet)
    mux.Post("/api/newfeature", a.handlePost)
}

func (a *API) handleGet(w http.ResponseWriter, r *http.Request) {
    param := r.URL.Query().Get("param")
    if param == "" {
        a.logger.Error("missing param")
        w.WriteHeader(http.StatusBadRequest)
        return
    }
    
    result, err := a.service.DoSomething(r.Context(), param)
    if err != nil {
        a.logger.Errorf("failed to process: %s", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    if err := json.NewEncoder(w).Encode(result); err != nil {
        a.logger.Errorf("failed to encode response: %s", err)
    }
}

func (a *API) handlePost(w http.ResponseWriter, r *http.Request) {
    // Similar pattern for POST
}

// NewAPI creates a new newfeature API handler.
func NewAPI(service ServiceInterface, logger logging.LoggerInterface) *API {
    return &API{
        service: service,
        logger:  logger,
    }
}
```

### Step 4: Wire into Router

Edit `pkg/web/router.go`:

```go
// Add import
import "github.com/canonical/identity-platform-login-ui/pkg/newfeature"

// In NewRouter function, after other services are initialized:
newFeatureService := newfeature.NewService(
    kratosPublicClient,  // or whatever clients you need
    tracer,
    monitor,
    logger,
)
newFeatureAPI := newfeature.NewAPI(newFeatureService, logger)
newFeatureAPI.RegisterEndpoints(mux)
```

### Step 5: Add Tests

Create `pkg/newfeature/service_test.go`:

```go
// Copyright 2025 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0

package newfeature

import (
    "context"
    "testing"
    
    "go.uber.org/mock/gomock"
    "go.opentelemetry.io/otel/trace"
)

//go:generate mockgen -build_flags=--mod=mod -package newfeature -destination ./mock_logger.go -source=../../internal/logging/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package newfeature -destination ./mock_tracing.go -source=../../internal/tracing/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package newfeature -destination ./mock_monitor.go -source=../../internal/monitoring/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package newfeature -destination ./mock_newfeature.go -source=./interfaces.go

func TestDoSomething(t *testing.T) {
    tests := []struct{
        name          string
        param         string
        mockSetup     func(*gomock.Controller) (*MockTracingInterface, *MockMonitorInterface, *MockLoggerInterface, *MockKratosClientInterface)
        expectedValue string
        expectedError error
    }{{
        name:  "success",
        param: "test",
        mockSetup: func(ctrl *gomock.Controller) (*MockTracingInterface, *MockMonitorInterface, *MockLoggerInterface, *MockKratosClientInterface) {
            mockTracer := NewMockTracingInterface(ctrl)
            mockMonitor := NewMockMonitorInterface(ctrl)
            mockLogger := NewMockLoggerInterface(ctrl)
            mockKratos := NewMockKratosClientInterface(ctrl)
            
            ctx := context.Background()
            mockTracer.EXPECT().Start(ctx, "newfeature.Service.DoSomething").
                Times(1).
                Return(ctx, trace.SpanFromContext(ctx))
            
            return mockTracer, mockMonitor, mockLogger, mockKratos
        },
        expectedValue: "test",
        expectedError: nil,
    }}
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctrl := gomock.NewController(t)
            defer ctrl.Finish()
            
            mockTracer, mockMonitor, mockLogger, mockKratos := tt.mockSetup(ctrl)
            service := NewService(mockKratos, mockTracer, mockMonitor, mockLogger)
            
            result, err := service.DoSomething(context.Background(), tt.param)
            
            if err != tt.expectedError {
                t.Errorf("expected error %v, got %v", tt.expectedError, err)
            }
            
            if result != nil && result.Value != tt.expectedValue {
                t.Errorf("expected value %s, got %s", tt.expectedValue, result.Value)
            }
        })
    }
}
```

Create `pkg/newfeature/handlers_test.go`:

```go
// Copyright 2025 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0

package newfeature

import (
    "net/http"
    "net/http/httptest"
    "testing"
    
    "github.com/go-chi/chi/v5"
    "go.uber.org/mock/gomock"
)

func TestHandleGet(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()
    
    mockService := NewMockServiceInterface(ctrl)
    mockLogger := NewMockLoggerInterface(ctrl)
    
    result := &Result{Value: "test"}
    mockService.EXPECT().
        DoSomething(gomock.Any(), "test").
        Return(result, nil)
    
    req := httptest.NewRequest(http.MethodGet, "/api/newfeature?param=test", nil)
    w := httptest.NewRecorder()
    
    mux := chi.NewMux()
    api := NewAPI(mockService, mockLogger)
    api.RegisterEndpoints(mux)
    
    mux.ServeHTTP(w, req)
    
    if w.Code != http.StatusOK {
        t.Errorf("expected status 200, got %d", w.Code)
    }
}
```

### Step 6: Generate Mocks

```bash
go generate ./...
# or specifically
go generate ./pkg/newfeature/...
```

### Step 7: Run Tests

```bash
make test
```

---

## Adding a New React Page (Frontend)

### Step 1: Create Page Component

Create `ui/pages/newpage.tsx`:

```typescript
import { FC, useEffect, useState } from "react";
import { useRouter } from "next/router";
import { MainTable, Spinner } from "@canonical/react-components";
import Layout from "../components/Layout";
import { fetchData } from "../api/newpage";

interface DataItem {
  id: string;
  name: string;
}

const NewPage: FC = () => {
  const router = useRouter();
  const [data, setData] = useState<DataItem[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  
  useEffect(() => {
    void fetchData()
      .then(setData)
      .catch((err) => {
        setError(err instanceof Error ? err.message : "Unknown error");
      });
  }, []);
  
  if (error) {
    return (
      <Layout title="Error">
        <p>Error: {error}</p>
      </Layout>
    );
  }
  
  if (!data) {
    return (
      <Layout title="Loading">
        <Spinner />
      </Layout>
    );
  }
  
  return (
    <Layout title="New Page">
      <MainTable
        headers={[
          { content: "ID" },
          { content: "Name" },
        ]}
        rows={data.map((item) => ({
          columns: [
            { content: item.id },
            { content: item.name },
          ],
        }))}
      />
    </Layout>
  );
};

export default NewPage;
```

### Step 2: Create API Utility

Create `ui/api/newpage.ts`:

```typescript
export interface DataItem {
  id: string;
  name: string;
}

export async function fetchData(): Promise<DataItem[]> {
  const response = await fetch("/api/newpage");
  
  if (!response.ok) {
    throw new Error(`Failed to fetch data: ${response.statusText}`);
  }
  
  return response.json();
}

export async function createItem(name: string): Promise<DataItem> {
  const response = await fetch("/api/newpage", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ name }),
  });
  
  if (!response.ok) {
    throw new Error(`Failed to create item: ${response.statusText}`);
  }
  
  return response.json();
}
```

### Step 3: Add E2E Test

Create `ui/tests/newpage.spec.ts`:

```typescript
import { test, expect } from "@playwright/test";

test("new page loads and displays data", async ({ page }) => {
  await page.goto("/newpage");
  
  // Wait for data to load
  await expect(page.getByRole("heading", { name: "New Page" })).toBeVisible();
  
  // Check table is visible
  await expect(page.getByRole("table")).toBeVisible();
  
  // Visual regression
  await expect(page).toHaveScreenshot({ 
    fullPage: true, 
    maxDiffPixels: 500,
  });
});

test("new page handles errors gracefully", async ({ page }) => {
  // Mock API to return error
  await page.route("/api/newpage", (route) => {
    void route.fulfill({
      status: 500,
      body: "Internal Server Error",
    });
  });
  
  await page.goto("/newpage");
  
  await expect(page.getByText(/Error:/)).toBeVisible();
  await expect(page).toHaveScreenshot({ 
    fullPage: true, 
    maxDiffPixels: 500,
  });
});
```

### Step 4: Run Tests

```bash
cd ui
npx playwright test newpage.spec.ts

# Or with UI
npx playwright test newpage.spec.ts --ui
```

---

## Adding Tracing to Existing Code

### Service Method Tracing

Add at the beginning of every public service method:

```go
func (s *Service) MyMethod(ctx context.Context, param string) (*Result, error) {
    ctx, span := s.tracer.Start(ctx, "package.Service.MyMethod")
    defer span.End()
    
    // Method implementation
}
```

**Format:** `"package.Service.Method"` (e.g., `"kratos.Service.GetLoginFlow"`)

### Testing with Tracer Mocks

Always expect tracer.Start() calls in tests:

```go
ctx := context.Background()
mockTracer.EXPECT().
    Start(ctx, "kratos.Service.GetLoginFlow").
    Times(1).
    Return(ctx, trace.SpanFromContext(ctx))
```

---

## Debugging Common Issues

### Backend: Mock Generation Fails

**Problem:** `make mocks` fails or mocks are out of date.

**Solution:**
```bash
# Clean old mocks
find . -name "mock_*.go" -delete

# Regenerate all mocks
go generate ./...

# Or just for one package
go generate ./pkg/kratos/...
```

### Backend: Test Fails with "unexpected call"

**Problem:** Gomock complains about unexpected method calls.

**Solution:**
- Check all mocked methods are expected
- Verify parameter matchers (use `gomock.Any()` for context)
- Ensure tracer.Start() is mocked
- Check Times(N) matches actual calls

```go
// If service calls kratos.GetFlow twice, expect it twice
mockKratos.EXPECT().GetFlow(gomock.Any(), flowID).Times(2)
```

### Frontend: TypeScript Errors After Adding New API

**Problem:** Type errors when using new API functions.

**Solution:**
- Define proper interfaces for request/response
- Export interfaces from API file
- Import and use in components

```typescript
// api/newpage.ts
export interface CreateItemRequest {
  name: string;
}

export interface CreateItemResponse {
  id: string;
  name: string;
}

// In component
import { CreateItemRequest, CreateItemResponse } from "../api/newpage";
```

### Frontend: Playwright Test Flaky

**Problem:** Test passes sometimes, fails other times.

**Solution:**
- Always use `await` for async operations
- Use `expect().toBeVisible()` to wait for elements
- Increase timeout if needed: `{ timeout: 10000 }`
- Check for race conditions in data loading

```typescript
// Bad - doesn't wait
page.getByRole("button").click();

// Good - waits for button to be ready
await page.getByRole("button").click();

// Better - wait for specific state
await expect(page.getByText("Loaded")).toBeVisible();
await page.getByRole("button").click();
```

### Build: `make build` Fails with "cmd/ui/dist not found"

**Problem:** Backend build expects frontend assets.

**Solution:**
```bash
# Build frontend first
make npm-build

# Then build backend
make build

# Or combined (sequential)
make npm-build build
```

**Never run in parallel:**
```bash
# Bad - race condition
make npm-build & make build

# Good - sequential
make npm-build build
```
