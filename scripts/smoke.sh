#!/usr/bin/env bash
set -euo pipefail

# quick smoke test: assumes docker compose is configured and migrations exist
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "1/5: bring up containers..."
docker compose up -d

echo "2/5: apply migrations (adjust file names if needed)..."
# apply all migrations you need; example:
docker exec -i agnos_postgres psql -U agnos -d agnos < migrations/001_create_patients.sql
docker exec -i agnos_postgres psql -U agnos -d agnos < migrations/002_create_staffs.sql
docker exec -i agnos_postgres psql -U agnos -d agnos < migrations/003_add_hospital_id_to_patients.sql || true
docker exec -i agnos_postgres psql -U agnos -d agnos < migrations/004_create_search_events.sql || true

echo "3/5: create staff (idempotent)"
curl -s -X POST http://localhost:8080/staff/create \
  -H "Content-Type: application/json" \
  -d '{"username":"smoke","password":"smoke123","hospital_id":"HIS-1","display_name":"Smoke Test"}' | jq .

echo "4/5: login to get token"
TOKEN=$(curl -s -X POST http://localhost:8080/staff/login \
  -H "Content-Type: application/json" \
  -d '{"username":"smoke","password":"smoke123","hospital_id":"HIS-1"}' | jq -r .access_token)

if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
  echo "ERROR: token not returned"
  exit 2
fi
echo " token ok"

echo "5/5: perform search and show result"
curl -s -H "Authorization: Bearer ${TOKEN}" -H "Content-Type: application/json" \
  -d '{"national_id":"N-1234567890","limit":1,"offset":0}' \
  http://localhost:8080/patient/search | jq .

echo "verify audit (recent rows):"
docker exec -it agnos_postgres psql -U agnos -d agnos -c \
"SELECT id, staff_id, hospital_id, result_count, created_at FROM search_events ORDER BY created_at DESC LIMIT 5;"
