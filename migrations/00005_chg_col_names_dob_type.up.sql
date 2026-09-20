DROP TABLE IF EXISTS students CASCADE;

CREATE TABLE students (
    admission_no  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    fname         VARCHAR(255) NOT NULL,
    mname         VARCHAR(255),
    lname         VARCHAR(255) NOT NULL,
    grade         VARCHAR(255) NOT NULL,
    dob           DATE NOT NULL,
    gender        VARCHAR(3) NOT NULL,
    nationality   VARCHAR(255) NOT NULL,
    address       VARCHAR(255) NOT NULL,
    guardian      VARCHAR(255) NOT NULL,
    g_contact     VARCHAR(255) NOT NULL,
    g_occupation  VARCHAR(255) NOT NULL,
    e_contact     VARCHAR(255) NOT NULL,
    med_con       VARCHAR(255) NOT NULL,
    allergies     VARCHAR(255) NOT NULL,
    photo         TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    grad_year     SMALLINT NOT NULL,
    seq_num       SMALLINT NOT NULL,

    UNIQUE (grad_year, seq_num)
);
