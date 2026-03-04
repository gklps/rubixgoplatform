-- State Consistency Verification
-- Returns one row per inconsistency between tokens and tokenchain.
-- Zero rows = consistent state.

SELECT
    t.token_id,
    t.transaction_id            AS tokens_tx,
    t.latest_position           AS tokens_pos,
    tc.transaction_id           AS tc_tx,
    tc.position                 AS tc_pos
FROM tokens t
LEFT JOIN tokenchain tc
    ON tc.token_id = t.token_id
    AND tc.position = t.latest_position
WHERE
    t.transaction_id IS DISTINCT FROM tc.transaction_id
    OR tc.token_id IS NULL
ORDER BY t.token_id;
