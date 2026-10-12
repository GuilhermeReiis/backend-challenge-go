BEGIN;

LOCK TABLE wallets, wager_transactions, wallet_ledger_entries
    IN ACCESS EXCLUSIVE MODE;

CREATE FUNCTION ledger_expected_direction(operation_kind TEXT, reference_kind TEXT)
RETURNS TEXT
LANGUAGE sql
IMMUTABLE
AS $$
    SELECT CASE
        WHEN operation_kind = 'BET' THEN 'DEBIT'
        WHEN operation_kind IN ('OPENING', 'WIN', 'REFUND') THEN 'CREDIT'
        WHEN operation_kind = 'ROLLBACK' AND reference_kind = 'BET' THEN 'CREDIT'
        WHEN operation_kind = 'ROLLBACK' AND reference_kind IN ('WIN', 'REFUND') THEN 'DEBIT'
        ELSE NULL
    END;
$$;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM wager_transactions AS operation
        JOIN wallets AS wallet ON wallet.id = operation.wallet_id
        WHERE operation.player_id IS DISTINCT FROM wallet.player_id
    ) THEN
        RAISE EXCEPTION 'existing operation does not belong to its wallet player'
            USING ERRCODE = '23503', CONSTRAINT = 'wager_wallet_player_currency_fk';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM wallet_ledger_entries AS entry
        JOIN wager_transactions AS operation ON operation.id = entry.transaction_id
        LEFT JOIN wager_transactions AS reference ON reference.id = operation.reference_transaction_id
        WHERE entry.direction IS DISTINCT FROM
            public.ledger_expected_direction(operation.kind, reference.kind)
    ) THEN
        RAISE EXCEPTION 'existing ledger direction does not match its operation'
            USING ERRCODE = '23514', CONSTRAINT = 'ledger_operation_direction_check';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM wallet_ledger_entries AS entry
        JOIN wager_transactions AS operation ON operation.id = entry.transaction_id
        WHERE entry.balance_after_minor IS DISTINCT FROM operation.result_balance_minor
    ) THEN
        RAISE EXCEPTION 'existing ledger balance does not match its operation result'
            USING ERRCODE = '23514', CONSTRAINT = 'ledger_operation_result_check';
    END IF;
END;
$$;

ALTER TABLE wallets
    ADD CONSTRAINT wallets_id_player_currency_unique
    UNIQUE (id, player_id, currency);

ALTER TABLE wager_transactions
    ADD CONSTRAINT wager_wallet_player_currency_fk
    FOREIGN KEY (wallet_id, player_id, currency)
    REFERENCES wallets (id, player_id, currency);

CREATE FUNCTION validate_ledger_operation()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    operation_kind TEXT;
    operation_result BIGINT;
    reference_id UUID;
    reference_kind TEXT;
    expected_direction TEXT;
BEGIN
    -- Acquire the wallet before operation/reference locks. Financial chain
    -- validation is added separately; this guard does not change the balance.
    PERFORM 1 FROM public.wallets WHERE id = NEW.wallet_id FOR UPDATE;

    SELECT kind, result_balance_minor, reference_transaction_id
        INTO operation_kind, operation_result, reference_id
        FROM public.wager_transactions
        WHERE id = NEW.transaction_id
        FOR SHARE;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'ledger operation does not exist'
            USING ERRCODE = '23503', CONSTRAINT = 'ledger_transaction_fk';
    END IF;

    IF operation_kind = 'ROLLBACK' THEN
        SELECT kind INTO reference_kind
            FROM public.wager_transactions
            WHERE id = reference_id
            FOR SHARE;
    END IF;

    expected_direction := public.ledger_expected_direction(operation_kind, reference_kind);

    IF NEW.direction IS DISTINCT FROM expected_direction THEN
        RAISE EXCEPTION 'ledger direction does not match its operation'
            USING ERRCODE = '23514', CONSTRAINT = 'ledger_operation_direction_check';
    END IF;

    IF NEW.balance_after_minor IS DISTINCT FROM operation_result THEN
        RAISE EXCEPTION 'ledger balance does not match its operation result'
            USING ERRCODE = '23514', CONSTRAINT = 'ledger_operation_result_check';
    END IF;

    RETURN NEW;
END;
$$;

CREATE TRIGGER ledger_validate_operation
    BEFORE INSERT ON wallet_ledger_entries
    FOR EACH ROW
    EXECUTE FUNCTION validate_ledger_operation();

COMMIT;
