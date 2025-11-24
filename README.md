# Agnos Search — README

Quick guide to run, test, and verify the project locally (Docker Compose).

## Prerequisites

* Docker & Docker Compose (or `docker compose`) installed
* `openssl` (optional, for generating JWT_SECRET)
* `go` (only if running locally or to run tests)

## Important files

* `docker-compose.yml` — compose config (Postgres + app)
* `migrations/` — SQL migrations (apply to Postgres)
* `internal/handler/patient_handler.go` — patient routes (search + GET)
* `internal/repository/analytics.go` — analytics repo (writes to `search_events`)

---

## 1) Start services (Docker Compose)

```bash
# from project root
docker compose up -d
```

If you added/changed `JWT_SECRET` in `docker-compose.yml`, rebuild to pick it up:

```bash
docker compose down
docker compose build --no-cache
docker compose up -d
```

## 2) Apply DB migrations

Run each SQL file in `migrations/` against the `agnos` database in the Postgres container:

```bash
# example: run migration 004_create_search_events.sql
docker exec -i agnos_postgres psql -U agnos -d agnos < migrations/004_create_search_events.sql
```

Verify migration:

```bash
docker exec -it agnos_postgres psql -U agnos -d agnos -c "\d+ search_events"
```

## 3) Create a staff user & login

Create user (example):

```bash
curl -s -X POST http://localhost:8080/staff/create \
  -H "Content-Type: application/json" \
  -d '{"username":"staff1","password":"password123","hospital_id":"HIS-1","display_name":"User 1"}' | jq
```

Login to get a token:

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/staff/login \
  -H "Content-Type: application/json" \
  -d '{"username":"staff1","password":"password123","hospital_id":"HIS-1"}' | jq -r .access_token)
```

## 4) Search patients (authenticated)

```bash
curl -i -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"national_id":"N-1234567890","limit":10,"offset":0}' \
  http://localhost:8080/patient/search | jq
```

## 5) Verify audit logs (search_events)

```bash
docker exec -it agnos_postgres psql -U agnos -d agnos -c \
"SELECT id, staff_id, hospital_id, result_count, created_at FROM search_events ORDER BY created_at DESC LIMIT 10;"
```

## 6) Run tests

```bash
# run all unit tests
go test ./...
```

## Environment

Important env vars (configure via `docker-compose.yml` or `.env`):

* `DATABASE_URL` — e.g. `postgres://agnos:secret@postgres:5432/agnos?sslmode=disable`
* `JWT_SECRET` — used to sign access tokens (set to a secure random value)
* `PORT` — HTTP port (default `8080`)

## Notes

* Tests use mocks for DB where applicable; passing `go test` does not guarantee the running container will work unless migrations have been applied and Postgres is up.
* For debugging, check the app logs: `docker logs -f agnos-search-app-1 --tail 200`.

