-- Forgot password: the user asks for a code by email, then sends it back with a new password.
-- One row per user (user_id is the primary key): asking again replaces the old code, so only the newest works.
CREATE TABLE IF NOT EXISTS password_resets (
    user_id     UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    code_hash   TEXT NOT NULL,                        -- sha256 of the code; the code itself only exists in the email
    attempts    SMALLINT NOT NULL DEFAULT 0,          -- wrong guesses so far; the code dies after 5
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),   -- when this code was sent; a new one waits a minute after it
    expires_at  TIMESTAMPTZ NOT NULL
);
