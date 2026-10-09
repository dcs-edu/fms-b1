-- recorded_by only ever held the staff member on cash payments, so online rows lose who started them.
UPDATE payments SET initiated_by = NULL WHERE reference IS NOT NULL;

ALTER TABLE payments RENAME COLUMN initiated_by TO recorded_by;
ALTER TABLE payments RENAME CONSTRAINT payments_initiated_by_fkey TO payments_recorded_by_fkey;
