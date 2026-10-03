CREATE TABLE IF NOT EXISTS utilities (
    id BIGSERIAL PRIMARY KEY,
    util_name VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true
);

CREATE TABLE IF NOT EXISTS utility_prices (
    bill_id BIGSERIAL PRIMARY KEY,
    util_id BIGINT NOT NULL REFERENCES utilities(id),
    sem VARCHAR(255) NOT NULL,                            -- semester name
    amount DECIMAL(10, 2) NOT NULL CHECK (amount > 0),
    UNIQUE (util_id, sem)
);

CREATE TABLE IF NOT EXISTS payments (
    txn_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bill_id BIGINT NOT NULL REFERENCES utility_prices(bill_id),
    admission_no UUID NOT NULL REFERENCES students(admission_no),
    amount DECIMAL(10, 2) NOT NULL CHECK (amount > 0),
    paid_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    reason VARCHAR(255)
);

-- Postgres does not index foreign key columns automatically
-- Semester reports join payments -> utility_prices on bill_id, so this is the column to index.
CREATE INDEX IF NOT EXISTS idx_payments_bill_id ON payments (bill_id);

-- Time-range reports (daily ledger, whole-semester table) filter on paid_at.
CREATE INDEX IF NOT EXISTS idx_payments_paid_at ON payments (paid_at);
