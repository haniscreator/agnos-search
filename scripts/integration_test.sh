#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "=== Integration test: start ==="

# 0) ensure .env exists
if [[ -n "${GITHUB_ACTIONS:-}" ]]; then
  echo "0/7: running in CI - writing Docker-friendly .env"
  cat > .env <<EOF
POSTGRES_USER=agnos
POSTGRES_PASSWORD=secret
POSTGRES_DB=agnos
PORT=8080
JWT_SECRET=ci-secret
HOSPITAL_BASE=http://hospital-a.api.co.th
# IMPORTANT: inside the app container, the Postgres host is 'postgres'
DATABASE_URL=postgres://agnos:secret@postgres:5432/agnos?sslmode=disable
EOF
else
  echo "0/7: .env already exists - using existing values"
fi

# 1) start containers
echo "1/7: docker compose up -d"
docker compose up -d

# 2) wait for postgres to be ready
echo "2/7: wait for postgres to be ready"
for i in {1..15}; do
  if docker exec -i agnos_postgres pg_isready -U agnos -d agnos >/dev/null 2>&1; then
    echo " Postgres is ready"
    break
  fi
  echo " Postgres not ready yet (attempt $i/15); sleeping 2s..."
  sleep 2
done

if ! docker exec -i agnos_postgres pg_isready -U agnos -d agnos >/dev/null 2>&1; then
  echo "ERROR: Postgres did not become ready in time"
  echo "--- docker logs agnos_postgres (tail 200) ---"
  docker logs agnos_postgres --tail 200 || true
  exit 1
fi

# 3) apply migrations (idempotent)
echo "3/7: apply migrations"
docker exec -i agnos_postgres psql -U agnos -d agnos < migrations/001_create_patients.sql
docker exec -i agnos_postgres psql -U agnos -d agnos < migrations/002_create_staffs.sql
docker exec -i agnos_postgres psql -U agnos -d agnos < migrations/003_add_hospital_id_to_patients.sql || true
docker exec -i agnos_postgres psql -U agnos -d agnos < migrations/004_create_search_events.sql || true

# 4) ensure test patient exists (upsert)
echo "4/7: insert or upsert test patient"
docker exec -i agnos_postgres psql -U agnos -d agnos -c "
INSERT INTO patients (
  id, patient_hn, national_id, passport_id,
  first_name_th, last_name_th,
  first_name_en, last_name_en,
  date_of_birth, phone_number, email, gender, raw_json, hospital_id
)
VALUES (
  '11111111-1111-1111-1111-111111111111',
  'HN-001',
  'N-1234567890',
  'P-ABC1234',
  'สมชาย',
  'ใจดี',
  'Somchai',
  'Jaidee',
  '1990-01-01',
  '0812345678',
  'somchai@example.com',
  'M',
  '{\"note\":\"seeded for tests\"}',
  'HIS-1'
)
ON CONFLICT (id) DO UPDATE SET national_id=EXCLUDED.national_id;
"

# 5) create staff (idempotent)
echo "5/7: create staff (idempotent)"
CREATE_RESP=$(curl -s -X POST http://localhost:8080/staff/create \
  -H "Content-Type: application/json" \
  -d '{"username":"itest_staff","password":"itest_pass","hospital_id":"HIS-1","display_name":"Integration Test"}')

echo "$CREATE_RESP" | jq . || echo "$CREATE_RESP"

# 6) login and get token
echo "6/7: login to get token"
LOGIN_RESP=$(curl -s -X POST http://localhost:8080/staff/login \
  -H "Content-Type: application/json" \
  -d '{"username":"itest_staff","password":"itest_pass","hospital_id":"HIS-1"}')

TOKEN=$(echo "$LOGIN_RESP" | jq -r .access_token 2>/dev/null || echo "")

if [[ -z "${TOKEN:-}" || "$TOKEN" == "null" ]]; then
  echo "ERROR: failed to obtain token"
  echo "login response was: $LOGIN_RESP"
  echo "--- docker logs agnos-search-app-1 (tail 200) ---"
  docker logs agnos-search-app-1 --tail 200 || true
  exit 2
fi
echo " token obtained"

# 7) run search and assert result
echo "7/7: perform search and validate response"
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

# verify audit row exists for this staff (strictly by staff)
echo "verifying audit row..."
# get staff id for itest_staff
STAFF_ID=$(docker exec -i agnos_postgres psql -U agnos -d agnos -t -c \
  "SELECT id FROM staffs WHERE username='itest_staff' AND hospital_id='HIS-1';" | tr -d '[:space:]' || true)
if [[ -z "$STAFF_ID" ]]; then
  echo "ERROR: itest_staff not found in staffs table; aborting audit assertion"
  exit 6
fi

AUDIT_COUNT_BY_STAFF=$(docker exec -i agnos_postgres psql -U agnos -d agnos -t -c \
  "SELECT count(1) FROM search_events WHERE hospital_id='HIS-1' AND staff_id='${STAFF_ID}';" | tr -d '[:space:]' || echo "0")
if [[ -z "$AUDIT_COUNT_BY_STAFF" ]]; then
  AUDIT_COUNT_BY_STAFF=0
fi

if [[ "$AUDIT_COUNT_BY_STAFF" -lt 1 ]]; then
  echo "ERROR: audit row for itest_staff not found (count=$AUDIT_COUNT_BY_STAFF)"
  exit 7
fi

echo "audit check: ok (by staff count=$AUDIT_COUNT_BY_STAFF)"
echo "=== Integration test: SUCCESS ==="
