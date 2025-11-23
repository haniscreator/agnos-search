-- migrations/001_create_patients.sql
CREATE TABLE IF NOT EXISTS patients (
  id UUID PRIMARY KEY,                       -- internal UUID
  patient_hn TEXT,                           -- hospital number (HN)
  national_id TEXT UNIQUE,                   -- national id (may be null)
  passport_id TEXT UNIQUE,                   -- passport id (may be null)
  first_name_th TEXT,
  middle_name_th TEXT,
  last_name_th TEXT,
  first_name_en TEXT,
  middle_name_en TEXT,
  last_name_en TEXT,
  date_of_birth DATE,
  phone_number TEXT,
  email TEXT,
  gender CHAR(1),                             -- 'M' or 'F'
  raw_json JSONB,                             -- store original raw response if desired
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now()
);

-- index to speed lookups by national_id / passport_id
CREATE INDEX IF NOT EXISTS idx_patients_national_id ON patients(national_id);
CREATE INDEX IF NOT EXISTS idx_patients_passport_id ON patients(passport_id);
