ALTER TABLE patients
  ADD COLUMN IF NOT EXISTS hospital_id TEXT;

CREATE INDEX IF NOT EXISTS idx_patients_hospital_id
  ON patients(hospital_id);
