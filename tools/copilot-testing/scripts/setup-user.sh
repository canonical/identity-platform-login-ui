#!/bin/bash
set -e

EMAIL=${1:-"test@example.com"}

# Delete if exists (optional, but good for cleanup, though Kratos IDs are UUIDs, so simple create is fine usually.
# But Kratos prevents duplicate emails usually. So we should probably check or just rely on 'create' to fail or succeed)
# For simplicity, we just try to create. If it fails due to conflict, that's fine, the user exists.

response=$(curl -s -w "%{http_code}" -X POST http://localhost:4434/admin/identities \
  -H "Content-Type: application/json" \
  -d '{
  "schema_id": "default",
  "traits": {
    "email": "'"$EMAIL"'"
  },
  "credentials": {
    "password": {
      "config": {
        "password": "Password123!"
      }
    }
  }
}')

http_code=${response: -3}

if [ "$http_code" -eq 201 ]; then
    echo "✅ User created: $EMAIL"
elif [ "$http_code" -eq 409 ]; then
    echo "ℹ️ User already exists: $EMAIL (proceeding)"
else
    echo "❌ Failed to create user. HTTP $http_code"
    echo "Response: $response"
    exit 1
fi
