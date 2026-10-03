DROP TABLE IF EXISTS book_issues CASCADE;
DROP TABLE IF EXISTS books CASCADE;
 
CREATE TABLE IF NOT EXISTS book_packs (
    pack_id       BIGSERIAL,
    amount        BIGINT        NOT NULL,
    stock_count   BIGINT,
    grade         VARCHAR(255)  NOT NULL,
    price         DECIMAL(10, 2),
    added_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (pack_id)
);
CREATE TABLE book_issues (
    issue_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    admission_no      UUID NOT NULL,
    pack_id           BIGINT NOT NULL,
    issued_at         TIMESTAMPTZ NOT NULL DEFAULT now(),

    FOREIGN KEY (admission_no) REFERENCES students(admission_no),
    FOREIGN KEY (pack_id) REFERENCES book_packs(pack_id)
);

ALTER TABLE students ALTER COLUMN gender TYPE VARCHAR(10); -- Fix the stupid mistake claude made. (No rollbacks for this fix)
