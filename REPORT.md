# Report: Findings on Agentic Driven Testing

I have explored four different approaches to implementing AI-driven QA agents for the Login UI.

## Context & Motivation

The Identity Platform Login UI is a critical component that interacts with multiple backend services (Kratos for identity, Hydra for OAuth2, OpenFGA for permissions). Testing is challenging because of the combinatorial explosion of user states:
*   **Configuration Matrix:** Features like MFA, WebAuthn, and Passwordless can be toggled on/off.
*   **User States:** Users can be active, locked out, have 1FA, 2FA, recovery codes, or expired sessions.
*   **Service Variations:** We need to validate behavior when backend services respond slowly or return unexpected errors.

Traditional E2E tests (Playwright) are excellent for regression but rigid. They only test the "happy paths" we explicitly script. The goal of this research was to build an "Agent" that can explore the UI dynamically, navigate complex state transitions, and introduce chaos to find edge cases we didn't predict.

Here is a summary of the attempts, the code produced, and the lessons learned.

## 0. The Gemini SDK Experiment (Python)

*Code not preserved in repo.*

My very first attempt involved building a Python-based agent using the Google Gemini SDK directly. Instead of using a standardized protocol like MCP, I provided the model with a set of custom Python functions (tools) to interact with the browser. The hypothesis was that a high-reasoning model could deduce how to test the app with zero prior knowledge of the DOM structure.

*   **How it works:** The LLM was given a system prompt and a list of callable tools (e.g., `click_element`, `get_page_text`). It would reason about the page state and invoke functions to proceed.
*   **Status:** Abandoned.
*   **Critique:**
    *   **Promising Logic:** The reasoning capabilities were excellent; it could understand flow nuances (e.g., "I need to register before I can login").
    *   **Rate Limits:** I hit the API rate limits almost immediately. A single test step might require multiple model round-trips (observe -> think -> act -> observe), freezing the agent constantly.
*   **Ideas for Improvement:**
    *   **Model Selection:** Creating a specialized "Browser Agent" model using Gemini 1.5 Flash (lower latency/cost).
    *   **Batching:** Updating the tools to allow the agent to queue multiple actions (e.g., "Fill email, fill password, click submit") in one API call to reduce round-trips.

## 1. The Heuristic Agent (`tools/chaos-agent`)

This was the first attempt at a "self-driving" browser agent using TypeScript. It uses a custom graph implementation to map URL patterns to "States" and decides on actions based on available transitions effectively creating a "crawler" for the app states.

*   **How it works:** It defines a `Graph` class (`src/core/graph.ts`) where nodes are page states (e.g., "Login Page", "Recovery Success"). It attempts to match the current browser URL to a state and executes a heuristic function to move forward.
*   **Status:** Fully functional for specific flows. Phase 1.1 (Recovery) and Phase 1.2 (WebAuthn) are implemented.
*   **Dev Experience:** The Playwright MCP was instrumental in writing these tests. It allowed Copilot to interact directly with the running web page to fix selector issues and logic errors in real-time, avoiding the slow cycle of traditional debugging.
*   **Critique:**
    *   **Maintenance Nightmare:** The "heuristic" logic (`src/heuristics/`) became complex quickly. Defining transitions manually for every possible edge case mimics the work of writing standard E2E tests but with less predictability.
    *   **Flaky by Design:** "Fuzzy" matching of states led to the agent sometimes getting lost or looping, which is hard to debug in CI.
*   **How to run:**
    ```bash
    cd tools/chaos-agent
    npm install
    # Run the "Headed" Exploration mode
    npm start -- --mode=exploration --duration=60
    # Run a specific flow
    npm run test:webauthn-register
    ```

## 2. The Copilot Interactive Session (`tools/copilot-testing`)

This approach scrapped the code-heavy agent in favor of using VS Code Copilot directly with the `@playwright/test` MCP server. The idea was to treat the AI as a manual QA tester that we "prompt" to explore the app interactively within the IDE.

*   **How it works:** A set of instructions (`README.md`, `FLOWS.md`) and helper scripts (`scripts/register-webauthn.js`) guide the AI. You open a chat session and tell Copilot: "Run the WebAuthn login flow."
*   **Status:** Concept only.
*   **Critique:**
    *   **Too Slow:** The latency of the MCP server + LLM roundtrips makes a single login take minutes.
    *   **Token Heavy:** Sending the accessibility tree back and forth consumes massive context.
    *   **Technical Limits:** Copilot via MCP cannot easily handle low-level browser protocols (CDP) required for WebAuthn virtualization. It failed to press the "virtual" security key.
*   **Ideas for Improvement:**
    *   **Optimized MCP:** Write a custom MCP server specifically for our UI that filters the DOM tree before sending it to Copilot, drastically reducing token usage.
    *   **Interactive Hooks:** Add "Human in the Loop" hooks where the AI can pause and ask the user to perform a hardware action (like touching a YubiKey) before resuming.

## 3. The XState Deterministic Agent (`tools/chaos-agent-xstate`)

This is the current "winner." I replaced the fuzzy custom graph from Approach #1 with **XState**, a dedicated state machine library. This separates the *model* (the map of the application) from the *execution* (Playwright). This approach allows us to mathematically prove that our tests cover specific transitions.

*   **How it works:** We define strict state machines (`src/machines/*.machine.ts`) for "Happy Paths" (e.g., `webauthn-login`). The agent follows this strict path but can be extended to take random "error" branches in the future.
*   **Key Features:**
    *   **Visualizer:** We can copy the machine definition into the [Stately Inspector](https://stately.ai/registry) to visually see the test coverage graph.
    *   **Accelerated Development:** By using the Playwright MCP, Copilot could "see" the page and write the machine definitions and selectors much faster than a human, effectively fixing issues by interacting with the page directly.
    *   **Video Recording:** Native integration with Playwright's recording capabilities for debugging failed runs.
    *   **CDP Integration:** Successfully mocks hardware tokens using Chrome DevTools Protocol.
*   **Status:** Success. Fast, stable, and containerizable.
    *   Supports `webauthn-login`, `totp-login`, and `recovery-login`.
*   **Critique:**
    *   **Rigid (for now):** Currently, it only runs happy paths.
    *   **Boilerplate:** heavily relies on TypeScript definitions for events and context.
*   **Ideas for Improvement:**
    *   **Chaos Transitions:** Inject probabilistic "Chaos Edges" into the graph (e.g., a 10% chance to disconnect the network or double-click a submit button) to stress-test the UI.
    *   **Model-Based Testing:** Use the XState graph to automatically generate thousands of path permutations instead of hand-coding just the "Happy Path".

*   **How to run:**
    ```bash
    cd tools/chaos-agent-xstate
    npm install
    npm start -- --scenario=webauthn-login --headed
    ```

## Bugs Uncovered

During this process, the agentic scrutiny revealed actual issues:

1.  **MFA Switch Visibility:** When both TOTP and WebAuthn are enabled, the link to switch methods on the challenge screen can sometimes be missing or unreachable.
2.  **WebAuthn/Sequencing Incompatibility:** It is not possible to use WebAuthn for 2FA if the sequencing flag is not enabled.

## Recommendation

Abandon **Attempt 0** (Pure SDK) as we do not have a license.
Abandon **Attempt 2** (Interactive MCP) due to cost and latency issues in their current form.
Freeze **Attempt 1** (Heuristic Agent) as the maintenance burden is too high.

Focus strictly on **Attempt 3 (XState)**. It provides the structure of E2E tests with the flexibility to model complex user journeys as a graph. This allows us to structurally guarantee coverage of the complex Kratos/Hydra matrix while retaining the ability to inject randomness later.
