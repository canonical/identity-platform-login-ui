#!/bin/bash
set -e

# Scope: If FILE is provided, focus on it. Otherwise, broader project scope.
FILE=$1

echo "🤖 QA Persona: Lead Architect & Security Researcher"
echo "----------------------------------------------------------------"
echo "I have generated a deep-dive prompt for Copilot."
echo "This prompt leverages Copilot's training data on Ory Kratos/Hydra"
echo "and instructs it to analyze your local codebase context."
echo "----------------------------------------------------------------"
echo ""
echo "COPY THE TEXT BELOW INTO COPILOT CHAT:"
echo "----------------------------------------------------------------"
echo "I need a comprehensive security and stability review of my Identity Platform Login UI."
echo "Stack: Go (Chi), Next.js, Ory Kratos, Ory Hydra, OpenFGA."
echo ""
if [ -n "$FILE" ]; then
    echo "Focus your analysis specifically on this file:"
    echo "@$FILE"
else
    echo "Please analyze my workspace, specifically looking at:"
    echo "- 'pkg/': Public handlers and interfaces."
    echo "- 'internal/': Core logic and integrations."
    echo "- 'ui/src': Frontend flow logic."
fi
echo ""
echo "TASK: Identify potential issues by cross-referencing my code with the official architecture and common pitfalls of Ory Kratos and Ory Hydra."
echo ""
echo "Specifically look for:"
echo "1. **Flow State Desynchronization**: Race conditions between the UI and Kratos's strict flow state (CSRF/Flow ID expiry)."
echo "2. **Error Swallowing**: Places in 'pkg/' where we might return 500s instead of user-friendly Kratos error messages."
echo "3. **Security Gaps**: Missing CSRF checks, improper cookie handling (compare with Kratos 'cookie_manager.go' patterns), or weak TOTP implementations."
echo "4. **WebAuthn Edge Cases**: Issues with 'navigator.credentials' handling in the frontend that might hang the UI."
echo ""
echo "Based on this analysis, propose 3 complex 'Chaos Scenarios' that could break this integration, which I should automate."
echo "----------------------------------------------------------------"
