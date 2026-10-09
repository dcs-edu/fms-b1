DROP INDEX IF EXISTS idx_payments_created_at;

-- The old table only held real payments, so attempts that never completed can't be kept.
DELETE FROM payments WHERE status <> 'completed';

ALTER TABLE payments
    DROP CONSTRAINT payments_paid_when_completed,
    DROP COLUMN recorded_by,
    DROP COLUMN created_at,
    DROP COLUMN status,
    DROP COLUMN reference,
    ADD COLUMN reason VARCHAR(255),
    ALTER COLUMN paid_at SET DEFAULT now(),
    ALTER COLUMN paid_at SET NOT NULL;
