-- migrations/002_create_staffs.sql
CREATE TABLE IF NOT EXISTS staffs (
  id UUID PRIMARY KEY,
  username TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  hospital_id TEXT NOT NULL,
  display_name TEXT,
  role TEXT DEFAULT 'staff',
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_staffs_hospital_id ON staffs(hospital_id);
