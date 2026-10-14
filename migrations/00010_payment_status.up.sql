-- payments now records attempts as well as money received:
--   pending   = parent sent to Paystack, no answer yet
--   completed = money received (online, or cash recorded by staff)
--   failed    = Paystack reported the charge failed or abandoned; kept as evidence
-- Anything that adds up money must filter status = 'completed'.
ALTER TABLE payments
    ADD COLUMN reference   TEXT UNIQUE,                         -- Paystack reference; NULL for cash
    ADD COLUMN status      VARCHAR(10) NOT NULL DEFAULT 'pending'
                           CONSTRAINT payments_status_check CHECK (status IN ('pending', 'completed', 'failed')),
    ADD COLUMN created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),  -- when the attempt started
    ADD COLUMN recorded_by UUID REFERENCES users(id),           -- staff who recorded a cash payment; NULL for online
    ALTER COLUMN paid_at DROP NOT NULL,                         -- now only set once money arrives
    ALTER COLUMN paid_at DROP DEFAULT,
    DROP COLUMN reason;

-- Every row that existed before this migration was a real payment.
UPDATE payments SET status = 'completed', created_at = paid_at;

-- Added after the backfill, since it would have rejected the old rows while they were still 'pending'.
-- A row has paid_at exactly when it is completed, so the two can never disagree.
ALTER TABLE payments
    ADD CONSTRAINT payments_paid_when_completed CHECK ((status = 'completed') = (paid_at IS NOT NULL));

-- Payment history ranges now filter on created_at, so failed and pending attempts show up too.
CREATE INDEX IF NOT EXISTS idx_payments_created_at ON payments (created_at);
