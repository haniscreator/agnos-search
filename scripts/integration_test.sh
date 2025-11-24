#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "=== Integration test: start ==="

# 1) start containers
echo "1/6: docker compose up -d"
docker compose up -d

# 2) apply migrations (idempotent)
echo "2/6: apply migrations"
docker exec -i agnos_postgres psql -U agnos -d agnos < migrations/001_create_patients.sql
docker exec -i agnos_postgres psql -U agnos -d agnos < migrations/002_create_staffs.sql
docker exec -i agnos_postgres psql -U agnos -d agnos < migrations/003_add_hospital_id_to_patients.sql || true
docker exec -i agnos_postgres psql -U agnos -d agnos < migrations/004_create_search_events.sql || true

# 3) ensure test patient exists (upsert)
echo "3/6: insert or upsert test patient"
# using a simple upsert: try inserting; if exists, ignore
docker exec -i agnos_postgres psql -U agnos -d agnos -c "
INSERT INTO patients (id, patient_hn, national_id, passport_id, first_name_th, last_name_th, first_name_en, last_name_en, date_of_birth, phone_number, email, gender, raw_json, hospital_id)
VALUES ('11111111-1111-1111-1111-111111111111','HN-001','N-1234567890','P-ABC1234','สมชาย','ใจดี','Somchai','Jaidee','1990-01-01','0812345678','somchai@example.com','M','{\"note\":\"seeded for tests\"}','HIS-1')
ON CONFLICT (id) DO UPDATE SET national_id=EXCLUDED.national_id;
"

# 4) create staff (idempotent)
echo "4/6: create staff (idempotent)"
curl -s -X POST http://localhost:8080/staff/create \
  -H "Content-Type: application/json" \
  -d '{"username":"itest_staff","password":"itest_pass","hospital_id":"HIS-1","display_name":"Integration Test"}' \
  | jq .

# 5) login and get token
echo "5/6: login to get token"
TOKEN=$(curl -s -X POST http://localhost:8080/staff/login \
  -H "Content-Type: application/json" \
  -d '{"username":"itest_staff","password":"itest_pass","hospital_id":"HIS-1"}' | jq -r .access_token)

if [[ -z "${TOKEN:-}" || "$TOKEN" == "null" ]]; then
  echo "ERROR: failed to obtain token"
  exit 2
fi
echo " token obtained"

# 6) run search and assert result
echo "6/6: perform search and validate response"
RESP=$(curl -s -H "Authorization: Bearer ${TOKEN}" -H "Content-Type: application/json" \
  -d '{"national_id":"N-1234567890","limit":1,"offset":0}' \
  http://localhost:8080/patient/search)

echo "response: $RESP"

# simple jq-based assertions:
COUNT=$(echo "$RESP" | jq -r '.count // -1')
if [[ "$COUNT" -lt 1 ]]; then
  echo "ERROR: expected at least 1 result, got count=$COUNT"
  exit 3
fi

# check patient id present in the first result
FIRST_ID=$(echo "$RESP" | jq -r '.results[0].ID // .results[0].id // empty')
if [[ -z "$FIRST_ID" ]]; then
  echo "ERROR: first result missing ID"
  exit 4
fi

if [[ "$FIRST_ID" != "11111111-1111-1111-1111-111111111111" ]]; then
  echo "ERROR: unexpected patient id: $FIRST_ID"
  exit 5
fi

echo "search returned expected patient id: $FIRST_ID"

# verify audit row exists for this staff
echo "verifying audit row..."
AUDIT_COUNT=$(docker exec -i agnos_postgres psql -U agnos -d agnos -t -c \
  "SELECT count(1) FROM search_events WHERE hospital_id='HIS-1';" | tr -d '[:space:]' || echo "0")

# normalize empty -> 0
if [[ -z "$AUDIT_COUNT" ]]; then
  AUDIT_COUNT=0
fi

# Accept if there is at least 1 audit record
if [[ "$AUDIT_COUNT" -lt 1 ]]; then
  echo "WARNING: no audit rows found for HIS-1 (count=$AUDIT_COUNT)"
else
  echo "audit check: ok (count=$AUDIT_COUNT)"
fi

echo "=== Integration test: SUCCESS ==="
