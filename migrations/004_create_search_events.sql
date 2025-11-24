-- migrations/004_create_search_events.sql
-- create extension if missing for uuid generation (safe id creation)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS search_events (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  staff_id TEXT,
  hospital_id TEXT,
  filters JSONB,
  result_count INTEGER,
  created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_search_events_hospital_id ON search_events(hospital_id);
