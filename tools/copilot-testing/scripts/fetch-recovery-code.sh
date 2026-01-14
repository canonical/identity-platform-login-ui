#!/bin/bash
set -e

EMAIL=${1:-"recovery-agent@example.com"}

# Fetch emails from MailSlurper
json_data=$(curl -s http://localhost:4437/mail)

# Extract the body of the latest email for the user
# dependency: jq
latest_body=$(echo "$json_data" | jq -r --arg email "$EMAIL" '
  .mailItems
  | map(select(.toAddresses[] | contains($email)))
  | sort_by(.dateSent)
  | reverse
  | .[0]
  | .body
')

if [ "$latest_body" == "null" ] || [ -z "$latest_body" ]; then
    echo "❌ No email found for $EMAIL"
    exit 1
fi

# Extract 6-digit code
code=$(echo "$latest_body" | grep -oE '[0-9]{6}' | head -n 1)

if [ -z "$code" ]; then
    echo "❌ Code not found in email body"
    echo "Body: $latest_body"
    exit 1
fi

echo "$code"
