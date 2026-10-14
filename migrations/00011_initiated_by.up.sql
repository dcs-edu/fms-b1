-- recorded_by only meant "staff who took cash", so online payments had nobody in it and you couldn't tell
-- whether a parent or the bursar started one. initiated_by = the user who started the payment, for cash
-- AND online. Whether it was cash or online is already told by reference (NULL = cash).
-- Online rows from before this migration stay NULL: nobody saved who started them.
ALTER TABLE payments RENAME COLUMN recorded_by TO initiated_by;
-- the foreign key keeps its old auto-generated name unless renamed; error messages show this name
ALTER TABLE payments RENAME CONSTRAINT payments_recorded_by_fkey TO payments_initiated_by_fkey;
