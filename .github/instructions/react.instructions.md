# TypeScript/React Conventions - Complete Reference

This document provides comprehensive TypeScript and React standards for the identity-platform-login-ui frontend.

## Component Structure

### Functional Components Only

**Always use functional components with hooks:**
- Never create class components
- Use `FC<Props>` type from React

```typescript
import { FC } from "react";

// Good
interface NodeInputProps {
  node: UiNode;
  value?: string;
  setValue?: (value: string) => void;
  disabled?: boolean;
}

export const NodeInput: FC<NodeInputProps> = ({ node, value, setValue, disabled }) => {
  // Implementation
};

// Bad - class component
export class NodeInput extends React.Component<NodeInputProps> {
  render() {
    // ...
  }
}
```

### Props

**Always destructure props in function signature:**
```typescript
// Good
export const NodeInput: FC<NodeInputProps> = ({ node, value, setValue }) => {
  // Use node, value, setValue directly
};

// Bad
export const NodeInput: FC<NodeInputProps> = (props) => {
  // Access via props.node, props.value, etc.
};
```

**Use TypeScript interfaces for prop types:**
```typescript
interface NodeInputProps {
  node: UiNode;              // Required
  value?: string;            // Optional - note the ?
  setValue?: (value: string) => void;
  disabled?: boolean;
}
```

**Mark optional props with `?`** - don't use `| undefined`.

### Imports

**Order:**
1. React imports
2. Third-party libraries
3. Local imports

```typescript
// Good
import React, { FC, useState } from "react";
import { Button, Input } from "@canonical/react-components";
import { NodeInputProps } from "./helpers";
import { formatDate } from "../util/date";

// Bad - mixed order
import { formatDate } from "../util/date";
import React from "react";
import { Button } from "@canonical/react-components";
```

**Use named imports**, avoid default exports for utilities:
```typescript
// Good
export const formatDate = (date: Date): string => { /* ... */ };
import { formatDate } from "./util";

// Acceptable for components
export default MyComponent;
```

## Naming Conventions

### Files

- **Components**: PascalCase matching component name
  - `NodeInput.tsx`, `UseOtherButton.tsx`
- **Utilities**: camelCase
  - `formatDate.ts`, `apiHelpers.ts`
- **Tests**: `.spec.ts` extension for Playwright E2E
  - `login.spec.ts`, `reset-password.spec.ts`

### Variables

- **camelCase** for variables and functions
  - `const flowId = "abc123";`
  - `function fetchLoginFlow() { }`
- **PascalCase** for components and types/interfaces
  - `interface NodeInputProps { }`
  - `const NodeInput: FC = () => { }`
- **UPPER_SNAKE_CASE** for constants
  - `const USER_EMAIL = "test@example.com";`
  - `const MAX_RETRY_COUNT = 3;`

## TypeScript Standards

### Strict Typing

**Enable strict mode** in `tsconfig.json` (already configured):
```json
{
  "compilerOptions": {
    "strict": true
  }
}
```

**No `any` types** - use `unknown` if type is truly unknown:
```typescript
// Bad
function processData(data: any) {
  return data.value;
}

// Good - use unknown and narrow
function processData(data: unknown) {
  if (typeof data === "object" && data !== null && "value" in data) {
    return (data as { value: string }).value;
  }
  throw new Error("Invalid data");
}

// Better - define proper interface
interface DataWithValue {
  value: string;
}

function processData(data: DataWithValue) {
  return data.value;
}
```

**Define interfaces for all props and data structures:**
```typescript
// Good
interface LoginFlowResponse {
  id: string;
  ui: {
    nodes: UiNode[];
    action: string;
  };
}

async function fetchLoginFlow(id: string): Promise<LoginFlowResponse> {
  const response = await fetch(`/api/kratos/login/flows/${id}`);
  return response.json();
}

// Bad - no typing
async function fetchLoginFlow(id) {
  const response = await fetch(`/api/kratos/login/flows/${id}`);
  return response.json();
}
```

**Use type inference where possible:**
```typescript
// Good - type inferred
const count = 5;
const message = "Hello";
const items = [1, 2, 3];

// Unnecessary - type is obvious
const count: number = 5;
const message: string = "Hello";
```

**Explicit types where clarity is needed:**
```typescript
// Good - return type makes intent clear
function calculateTotal(items: Item[]): number {
  return items.reduce((sum, item) => sum + item.price, 0);
}

// Function parameters should always have types
function processItem(item: Item, index: number): void {
  // ...
}
```

### Async/Await

**Always use `async/await`** for asynchronous operations:
```typescript
// Good
async function fetchData(id: string): Promise<Data> {
  const response = await fetch(`/api/data/${id}`);
  if (!response.ok) {
    throw new Error(`Failed to fetch: ${response.statusText}`);
  }
  return response.json();
}

// Bad - promise chains
function fetchData(id: string): Promise<Data> {
  return fetch(`/api/data/${id}`)
    .then(response => {
      if (!response.ok) {
        throw new Error(`Failed to fetch: ${response.statusText}`);
      }
      return response.json();
    });
}
```

**Use `void` operator for fire-and-forget:**
```typescript
// Good - intentionally not awaiting
void router.push("/login");

// Bad - missing await is likely a bug
router.push("/login");  // ESLint will warn
```

**Handle errors with try/catch:**
```typescript
async function handleSubmit(data: FormData): Promise<void> {
  try {
    const result = await submitForm(data);
    setSuccess(result);
  } catch (error) {
    if (error instanceof Error) {
      setError(error.message);
    } else {
      setError("An unknown error occurred");
    }
  }
}
```

## ESLint and Prettier

### Linting

**Commands:**
- `npm run lint` - check for errors/warnings
- `npm run fix-lint` - auto-fix issues

**No warnings or errors allowed in production code:**
```typescript
// Bad - ESLint will error
const unused = 5;  // Error: 'unused' is assigned but never used

// Bad - ESLint will error
console.log("Debug");  // Error: Unexpected console statement

// Good - remove unused code and debug statements
```

**Disable rules sparingly** and only with justification:
```typescript
// Acceptable with comment explaining why
// eslint-disable-next-line @typescript-eslint/no-explicit-any
function legacyWrapper(data: any) {
  // This wraps a third-party library that doesn't have types
}

// Bad - no explanation
// eslint-disable-next-line @typescript-eslint/no-explicit-any
function myFunction(data: any) {
  // ...
}
```

### Prettier Formatting

**Configuration (already set up):**
- 2-space indentation
- Single quotes for strings
- Trailing commas in objects/arrays
- Semicolons required

**Prettier runs automatically** - don't fight it:
```typescript
// Prettier will format to this
const config = {
  name: "example",
  value: 42,
};

// Not this (missing trailing comma)
const config = {
  name: "example",
  value: 42
};
```

## React Hooks

### useState

```typescript
import { useState, FC } from "react";

const MyComponent: FC = () => {
  // Type is inferred
  const [count, setCount] = useState(0);
  
  // Explicit type when needed
  const [data, setData] = useState<Data | null>(null);
  
  return (
    <div>
      <button onClick={() => setCount(count + 1)}>Count: {count}</button>
    </div>
  );
};
```

### useEffect

```typescript
import { useEffect, FC } from "react";

const MyComponent: FC<{ id: string }> = ({ id }) => {
  useEffect(() => {
    // Effect runs when id changes
    void fetchData(id);
  }, [id]);  // Dependencies array
  
  // Cleanup
  useEffect(() => {
    const subscription = subscribeToData();
    return () => {
      subscription.unsubscribe();  // Cleanup function
    };
  }, []);
  
  return <div>...</div>;
};
```

### Custom Hooks

```typescript
// Use "use" prefix
function useFormState<T>(initialValue: T) {
  const [value, setValue] = useState(initialValue);
  const [errors, setErrors] = useState<string[]>([]);
  
  const validate = () => {
    // Validation logic
  };
  
  return { value, setValue, errors, validate };
}

// Usage
const MyForm: FC = () => {
  const form = useFormState({ email: "", password: "" });
  // ...
};
```

## Testing with Playwright

### Test Structure

```typescript
import { test, expect } from "@playwright/test";

test("descriptive test name", async ({ page, context }) => {
  // Arrange
  await page.goto("/login");
  
  // Act
  await page.getByLabel("Email").fill("user@example.com");
  await page.getByLabel("Password").fill("password123");
  await page.getByRole("button", { name: "Sign in" }).click();
  
  // Assert
  await expect(page.getByText("Welcome")).toBeVisible();
  await expect(page).toHaveScreenshot({ fullPage: true, maxDiffPixels: 500 });
});
```

### Best Practices

**Use Page Object pattern** for reusable utilities:
```typescript
// ui/tests/helpers/login.ts
export const USER_EMAIL = "test@example.com";
export const USER_PASSWORD = "Test1234!";

export async function userPassLogin(page: Page, email: string, password: string) {
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password").fill(password);
  await page.getByRole("button", { name: "Sign in" }).click();
}

// In test file
import { userPassLogin, USER_EMAIL, USER_PASSWORD } from "./helpers/login";

test("login flow", async ({ page }) => {
  await page.goto("/login");
  await userPassLogin(page, USER_EMAIL, USER_PASSWORD);
  // ...
});
```

**Always use `await`** for async operations:
```typescript
// Good
await page.getByRole("button").click();
await expect(page.getByText("Success")).toBeVisible();

// Bad - missing await
page.getByRole("button").click();  // Returns promise, doesn't wait
expect(page.getByText("Success")).toBeVisible();  // May fail due to timing
```

**Use visual regression testing:**
```typescript
await expect(page).toHaveScreenshot({ 
  fullPage: true, 
  maxDiffPixels: 500  // Allow small differences (anti-aliasing, fonts)
});
```

**Clean up test data:**
```typescript
import { resetIdentities } from "./helpers/kratosIdentities";

test("create account", async ({ page }) => {
  resetIdentities();  // Clean slate before test
  
  // Test code...
});
```

## Common Patterns

### API Calls

```typescript
// ui/api/flows.ts
export async function createLoginFlow(refresh?: string): Promise<LoginFlow> {
  const params = new URLSearchParams();
  if (refresh) {
    params.set("refresh", refresh);
  }
  
  const response = await fetch(`/api/kratos/login/flows?${params}`);
  if (!response.ok) {
    throw new Error(`Failed to create login flow: ${response.statusText}`);
  }
  
  return response.json();
}

// In component
const MyComponent: FC = () => {
  const [flow, setFlow] = useState<LoginFlow | null>(null);
  
  useEffect(() => {
    void createLoginFlow().then(setFlow);
  }, []);
  
  return flow ? <FlowForm flow={flow} /> : <Loading />;
};
```

### Form Handling

```typescript
import { FC, FormEvent } from "react";

const LoginForm: FC = () => {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  
  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    
    try {
      await submitLogin({ email, password });
    } catch (error) {
      console.error("Login failed", error);
    }
  };
  
  return (
    <form onSubmit={handleSubmit}>
      <Input
        type="email"
        label="Email"
        value={email}
        onChange={(e) => setEmail(e.target.value)}
      />
      <Input
        type="password"
        label="Password"
        value={password}
        onChange={(e) => setPassword(e.target.value)}
      />
      <Button type="submit">Sign in</Button>
    </form>
  );
};
```

### Error Boundaries

```typescript
import { Component, ErrorInfo, ReactNode } from "react";

interface Props {
  children: ReactNode;
}

interface State {
  hasError: boolean;
}

class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false };
  }
  
  static getDerivedStateFromError(): State {
    return { hasError: true };
  }
  
  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error("Caught error:", error, errorInfo);
  }
  
  render() {
    if (this.state.hasError) {
      return <h1>Something went wrong.</h1>;
    }
    
    return this.props.children;
  }
}
```

Note: Error boundaries are one of the **only** acceptable uses of class components in modern React.
