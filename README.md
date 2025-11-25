# Agnos Search
[![CI](https://github.com/haniscreator/agnos-search/actions/workflows/ci.yml/badge.svg)](https://github.com/haniscreator/agnos-search/actions/workflows/ci.yml)

End-to-end guide to run, test, and verify the Agnos Search service locally using Docker Compose, with proper environment variables, migrations, authentication, and audit logging.

## Features
- Patient search (with filters & hospital scoping)
- Staff registration & login (JWT-based)
- JWT-protected API endpoints
- Search auditing → writes to search_events
- Dockerized Postgres + Go service
- Full smoke test script (scripts/smoke.sh)
- .env support (loaded automatically in container)


## 🚀 1. Setup
1.1 Prerequisites
- Docker & Docker Compose
- Go (optional — if running locally or for tests)
- openssl (optional — for generating JWT secret)

## 📦 2. Environment Setup
This project uses .env and .env.example.
.env.example
```bash
# --- Postgres ---
POSTGRES_USER=agnos
POSTGRES_PASSWORD=secret
POSTGRES_DB=agnos

# --- Application ---
PORT=8080
JWT_SECRET=your-secret-here
HOSPITAL_BASE=http://hospital-a.api.co.th

# DATABASE_URL for the app container
DATABASE_URL=postgres://agnos:secret@postgres:5432/agnos?sslmode=disable
```

## 🐳 3. Start the System (Docker Compose)
```bash
docker compose up -d
```
To rebuild (when Golang code changes):
```bash
docker compose down
docker compose build --no-cache
docker compose up -d
```
Check logs:
```bash
docker logs -f agnos-search-app-1
```
You should see:
```bash
Loaded environment .env (JWT_SECRET detected)
```

## 🗄️ 4. Run Database Migrations
Each SQL file in migrations/ should be applied manually.

Example:
```bash
docker exec -i agnos_postgres psql -U agnos -d agnos < migrations/001_create_patients.sql
```
Verify a migration:
```bash
docker exec -it agnos_postgres psql -U agnos -d agnos -c "\d+ search_events"
```

## 👤 5. Create Staff User
```bash
curl -s -X POST http://localhost:8080/staff/create \
  -H "Content-Type: application/json" \
  -d '{"username":"staff1","password":"password123","hospital_id":"HIS-1","display_name":"User 1"}' | jq
```

## 🔐 6. Login & Get JWT Token
```bash
TOKEN=$(curl -s -X POST http://localhost:8080/staff/login \
  -H "Content-Type: application/json" \
  -d '{"username":"staff1","password":"password123","hospital_id":"HIS-1"}' | jq -r .access_token)

echo $TOKEN
```

## 🔍 7. Search Patients (Authenticated)
```bash
curl -s -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"national_id":"N-1234567890","limit":10,"offset":0}' \
  http://localhost:8080/patient/search | jq
```
Expected response:
```bash
{
  "count": 1,
  "limit": 10,
  "offset": 0,
  "results": [ ... ]
}
```

## 🧾 8. View Search Audit Logs
```bash
docker exec -it agnos_postgres psql -U agnos -d agnos -c \
"SELECT id, staff_id, hospital_id, result_count, created_at FROM search_events ORDER BY created_at DESC LIMIT 10;"
```

## 🧪 9. Run Unit Tests
```bash
go test ./...
```

All tests should pass:
```bash
ok   internal/handler
ok   internal/repository
ok   internal/service
...
```

## 🔥 10. Smoke Test Script
You can run the entire workflow end-to-end with:
```bash
./scripts/smoke.sh
```
It performs:
1. Start containers
2. Apply migrations
3. Create staff
4. Login
5. Perform search
6. Check search_events table

Expected output is similar to:
```bash
5/5: perform search
{ ... patient result ... }
verify audit:
id | staff_id | hospital_id | result_count | created_at
```

## ✅ 11. CI Integration Test (GitHub Actions)
This project includes a full Docker-based integration test that runs automatically in GitHub Actions using:
```bash
./scripts/integration_test.sh
```
The integration test simulates the entire workflow inside CI:
1. Creates a .env optimized for Docker-in-CI
2. Starts Postgres + the Go API via docker compose
3. Applies migrations
4. Seeds a test patient
5. Creates staff user
6. Performs login (JWT)
7. Runs /patient/search
8. Validates the response and audit logs
9. Fails CI if any step fails

📌 Run integration test locally (optional)
You can run the same script on your machine:
```bash
./scripts/integration_test.sh
```

This lets you reproduce CI failures locally.

📄 View CI workflow
GitHub Actions workflow file:
```bash
.github/workflows/ci.yml
```
Live build badge:
```bash
[![CI](https://github.com/haniscreator/agnos-search/actions/workflows/ci.yml/badge.svg)](https://github.com/haniscreator/agnos-search/actions/workflows/ci.yml)

```

## 📁 Folder Structure
```bash
agnos-search/
├── cmd/
│   └── search/        # main.go entrypoint
├── internal/
│   ├── adapter/       # external hospital API
│   ├── db/            # pgx pool
│   ├── handler/       # gin handlers
│   ├── middleware/    # JWT auth
│   ├── repository/    # data layer
│   ├── service/       # business logic
│   └── ...
├── migrations/        # SQL migrations
├── scripts/
│   └── smoke.sh       # E2E validation
├── docker-compose.yml
├── .env.example
├── .env (ignored)
└── README.md
```

## 📝 Notes
- The app loads .env inside the Docker container.
- JWT_SECRET must not be empty or auth will fail.
- Tests mock DB connections → real DB required for runtime.
- Always rebuild containers after Go code changes.


