BEGIN;
CREATE TABLE organizations (
  name text PRIMARY KEY
);
COMMENT ON TABLE organizations IS 'Organizations using the service';
COMMENT ON COLUMN organizations.name IS 'Unique name of organization';
COMMIT;
