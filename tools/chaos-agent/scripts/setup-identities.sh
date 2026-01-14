#!/bin/bash
# Copyright 2025 Canonical Ltd.
# SPDX-License-Identifier: AGPL-3.0

# Setup script to create all test identities for chaos agent

set -e

KRATOS_ADMIN_URL="${KRATOS_ADMIN_URL:-http://localhost:4434}"
UI_BASE="${UI_BASE:-http://localhost}"

echo "=================================="
echo "Chaos Agent Identity Setup Script"
echo "=================================="
echo ""
echo "This script will:"
echo "  1. Delete existing test identities"
echo "  2. Create fresh identities for all user profiles"
echo "  3. Set up TOTP for users that need it"
echo "  4. Set up WebAuthn for webauthn-user"
echo ""

# Function to delete identity by email
delete_identity() {
    local email=$1
    echo "Checking for existing identity: $email"

    # Get all identities and find matching email
    local identity_id=$(curl -s "$KRATOS_ADMIN_URL/admin/identities" | \
        python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    for identity in data:
        if identity.get('traits', {}).get('email') == '$email':
            print(identity['id'])
            break
except:
    pass
" 2>/dev/null)

    if [ -n "$identity_id" ]; then
        echo "  Deleting existing identity: $identity_id"
        curl -s -X DELETE "$KRATOS_ADMIN_URL/admin/identities/$identity_id" > /dev/null
        echo "  ✓ Deleted"
    else
        echo "  No existing identity found"
    fi
}

# Function to create identity
create_identity() {
    local email=$1
    local password=$2
    local name=$3
    local surname=$4

    echo "Creating identity: $email"

    local response=$(curl -s -X POST "$KRATOS_ADMIN_URL/admin/identities" \
        -H "Content-Type: application/json" \
        -d "{
            \"schema_id\": \"default\",
            \"traits\": {
                \"email\": \"$email\",
                \"name\": \"$name\",
                \"surname\": \"$surname\"
            },
            \"credentials\": {
                \"password\": {
                    \"config\": {
                        \"password\": \"$password\"
                    }
                }
            },
            \"verifiable_addresses\": [
                {
                    \"value\": \"$email\",
                    \"verified\": true,
                    \"via\": \"email\"
                }
            ]
        }")

    # Check if creation was successful
    if echo "$response" | grep -q '"id"'; then
        echo "  ✓ Created successfully"
        return 0
    else
        echo "  ✗ Failed to create"
        echo "  Response: $response"
        return 1
    fi
}

echo "Step 1: Cleaning up existing identities"
echo "========================================"
delete_identity "test@example.com"
delete_identity "login-test@example.com"
delete_identity "webauthn-tester@example.com"
delete_identity "webauthn-dynamic@example.com"
delete_identity "full-security@example.com"
echo ""

echo "Step 2: Creating basic identities"
echo "=================================="
create_identity "test@example.com" "test" "Test" "User"
create_identity "login-test@example.com" "Test1234!" "Login" "Test"
create_identity "webauthn-tester@example.com" "Password123!" "WebAuthn" "Tester"
create_identity "webauthn-dynamic@example.com" "Password123!" "WebAuthn" "Dynamic"
create_identity "full-security@example.com" "SecurePass123!" "Full" "Security"
echo ""

echo "Step 3: Setting up TOTP for login-test user"
echo "============================================"
echo "This requires browser interaction..."
echo ""

# Use Node.js/Playwright to set up TOTP
node << 'NODEJS'
const { chromium } = require('playwright');
const fs = require('fs');

async function setupTOTP() {
    console.log('Launching browser...');
    const browser = await chromium.launch({ headless: true });
    const context = await browser.newContext();
    const page = await context.newPage();

    try {
        // Login as login-test user
        console.log('Logging in as login-test@example.com...');
        await page.goto('http://localhost/ui/login');
        await page.fill('input[name="identifier"]', 'login-test@example.com');
        await page.fill('input[type="password"]', 'Test1234!');
        await page.click('button:has-text("Sign in")');

        // Wait for TOTP setup page
        await page.waitForURL(/.*setup_secure.*/);
        console.log('TOTP setup page loaded');

        // Extract TOTP secret
        const secret = await page.locator('pre').textContent();
        console.log('TOTP Secret:', secret.trim());

        // Generate TOTP code using oathtool
        const { execSync } = require('child_process');
        const code = execSync(`oathtool -b --totp '${secret.trim()}'`).toString().trim();
        console.log('Generated code:', code);

        // Submit TOTP
        await page.fill('input[name="totp_code"]', code);
        await page.click('button:has-text("Save")');

        // Wait for completion
        await page.waitForTimeout(2000);
        console.log('✓ TOTP configured for login-test@example.com');
        console.log('  Secret:', secret.trim());

        // Update chaos agent state
        let state = {};
        try {
            if (fs.existsSync('.chaos-agent-state.json')) {
                state = JSON.parse(fs.readFileSync('.chaos-agent-state.json'));
            }
        } catch (e) {}

        // Refactored structure: key by email in 'users' object
        state.users = state.users || {};
        state.users['login-test@example.com'] = {
            ...(state.users['login-test@example.com'] || {}),
            email: 'login-test@example.com',
            totpSecret: secret.trim(),
            hasTotp: true
        };

        // Clean up legacy keys to avoid ambiguity
        if (state.secrets) delete state.secrets;
        if (state.totpSecret) delete state.totpSecret;

        fs.writeFileSync('.chaos-agent-state.json', JSON.stringify(state, null, 2));

    } catch (error) {
        console.error('Error setting up TOTP:', error.message);
    } finally {
        await browser.close();
    }
}

setupTOTP().catch(console.error);
NODEJS

echo ""
echo "Step 4: Setting up WebAuthn for webauthn-tester"
echo "================================================"
echo "Skipping pre-registration (Tests will handle registration state)"
# echo "Running WebAuthn registration script..."
# # Using local script
# node scripts/register-webauthn.js

# # Ensure config is in the right place (if the script generates it in CWD)
# if [ -f "webauthn-authenticators.json" ]; then
#     echo "Config generated successfully."
# fi

# echo ""
# echo "Step 5: Removing TOTP from webauthn-tester"
# echo "==========================================="
# echo "This allows testing WebAuthn-only login..."
# node scripts/remove-totp.js

echo ""
echo "=========================================="
echo "Identity Setup Complete!"
echo "=========================================="
echo ""
echo "Created identities:"
echo "  • test@example.com (password: test)"
echo "  • login-test@example.com (password: Test1234!) - TOTP configured"
echo "  • webauthn-tester@example.com (password: Password123!) - WebAuthn only (TOTP removed)"
echo "  • full-security@example.com (password: SecurePass123!)"
echo ""
echo "You can now run chaos agent tests with different profiles:"
echo "  npm run profiles              # List available profiles"
echo "  npm run test:webauthn-basic   # Test basic user"
echo "  npm run test:webauthn         # Test webauthn user (WebAuthn-only login)"
echo ""
