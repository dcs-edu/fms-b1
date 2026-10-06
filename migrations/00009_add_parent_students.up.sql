-- Many-to-many: a parent can have several children, a child can have two parents.
CREATE TABLE IF NOT EXISTS parent_students (
    parent_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    admission_no  UUID NOT NULL REFERENCES students(admission_no) ON DELETE CASCADE,
    linked_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (parent_id, admission_no)
);

-- The primary key already serves "children of this parent" (parent_id comes first).
-- This one serves the reverse: "parents of this student".
CREATE INDEX IF NOT EXISTS idx_parent_students_admission_no ON parent_students (admission_no);

-- Self-registration now creates parent accounts. Staff roles are only ever granted by an admin.
ALTER TABLE users ALTER COLUMN role SET DEFAULT 'parent';
