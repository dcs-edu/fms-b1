-- Postgres can't remove a value from an enum, so the type is rebuilt without them.
-- Anyone holding a removed role falls back to the old default.
UPDATE users SET role = 'teacher' WHERE role IN ('bursar', 'parent');

ALTER TABLE users ALTER COLUMN role DROP DEFAULT;
ALTER TYPE user_role RENAME TO user_role_old;
CREATE TYPE user_role AS ENUM ('admin', 'principal', 'teacher');
ALTER TABLE users ALTER COLUMN role TYPE user_role USING role::text::user_role;
ALTER TABLE users ALTER COLUMN role SET DEFAULT 'teacher';
DROP TYPE user_role_old;
