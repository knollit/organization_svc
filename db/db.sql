BEGIN;
CREATE TABLE organizations (
  name varchar(95) PRIMARY KEY
);
COMMENT ON TABLE organizations IS 'Organizations using the service';
COMMENT ON COLUMN organizations.name IS 'Unique name of organization';
COMMIT;
