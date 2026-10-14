-- New enum values must be committed before anything can use them,
-- which is why the table and default that need 'parent' live in the next migration.
ALTER TYPE user_role ADD VALUE IF NOT EXISTS 'bursar';
ALTER TYPE user_role ADD VALUE IF NOT EXISTS 'parent';
