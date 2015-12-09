BEGIN;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE TABLE IF NOT EXISTS organizations (
  id   BIGSERIAL PRIMARY KEY,
  name text UNIQUE
);
CREATE INDEX ON organizations (name);
COMMENT ON TABLE organizations IS 'Organizations using the service';
COMMENT ON COLUMN organizations.id IS 'UUID of organization';
COMMENT ON COLUMN organizations.name IS 'Unique name of organization';
COMMIT;
