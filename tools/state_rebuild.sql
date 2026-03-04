-- State Rebuild Procedure
-- Rebuilds the tokens table from authoritative tokenchain + transactions data.
-- Run inside a transaction. Safe to re-run (idempotent).

BEGIN;

-- Clear current token state
DELETE FROM tokens;

-- Rebuild token state from tokenchain (latest position per token)
INSERT INTO tokens (token_id, transaction_id, latest_position, latest_role)
SELECT DISTINCT ON (tc.token_id)
    tc.token_id,
    tc.transaction_id,
    tc.position       AS latest_position,
    tc.role           AS latest_role
FROM tokenchain tc
JOIN transactions t ON tc.transaction_id = t.id
ORDER BY tc.token_id, tc.position DESC;

COMMIT;
