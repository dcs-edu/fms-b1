CREATE TABLE students (
    fname        VARCHAR(255)   NOT NULL,
    mname        VARCHAR(255),
    lname        VARCHAR(255)   NOT NULL,
    grade        VARCHAR(255)   NOT NULL,
    dob          SMALLINT       NOT NULL,
    gender       VARCHAR(3)     NOT NULL,
    nationality  VARCHAR(255)   NOT NULL,
    address      VARCHAR(255)   NOT NULL,
    guardian     VARCHAR(255)   NOT NULL,
    gContact     VARCHAR(255)   NOT NULL,
    goccupation  VARCHAR(255)   NOT NULL,
    eContact     VARCHAR(255)   NOT NULL,
    medCon       VARCHAR(255)   NOT NULL,
    allergies    VARCHAR(255)   NOT NULL,
    photo        TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    grad_year    SMALLINT NOT NULL,
    seq_num      SMALLINT NOT NULL,
    admission_no  UUID UNIQUE DEFAULT gen_random_uuid(),

    PRIMARY KEY (grad_year, seq_num)
  );
  CREATE TABLE IF NOT EXISTS books (
    book_id       BIGSERIAL,
    stock_count   BIGINT,
    book_name     VARCHAR(255)  NOT NULL,
    subject       VARCHAR(255)  NOT NULL,
    grade         VARCHAR(255)  NOT NULL,
    price         DECIMAL(10, 2),
    added_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (book_id)
);
CREATE TABLE book_issues (
    issue_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_grad_year SMALLINT,
    student_seq_num   SMALLINT,
    book_id           BIGINT,
    issued_at         TIMESTAMPTZ NOT NULL DEFAULT now(),

    FOREIGN KEY (student_grad_year, student_seq_num) REFERENCES students(grad_year, seq_num),
    FOREIGN KEY (book_id) REFERENCES books(book_id)
);
