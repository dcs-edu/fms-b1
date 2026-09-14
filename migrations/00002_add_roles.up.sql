CREATE TYPE user_role AS ENUM ('admin', 'principal', 'teacher');

ALTER TABLE users ADD COLUMN role user_role NOT NULL DEFAULT 'teacher';
