ALTER TABLE wager_transactions
    ADD CONSTRAINT wager_ledger_target_unique
    UNIQUE (id, wallet_id, currency, amount_minor, status);

CREATE TABLE wallet_ledger_entries (
    id UUID PRIMARY KEY,
    sequence BIGINT GENERATED ALWAYS AS IDENTITY UNIQUE,

    wallet_id UUID NOT NULL,
    transaction_id UUID NOT NULL,
    transaction_status VARCHAR(20) NOT NULL DEFAULT 'PROCESSED',

    direction VARCHAR(6) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    amount_minor BIGINT NOT NULL,
    balance_before_minor BIGINT NOT NULL,
    balance_after_minor BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT ledger_wallet_currency_fk
        FOREIGN KEY (wallet_id, currency)
        REFERENCES wallets (id, currency),

    CONSTRAINT ledger_transaction_fk
        FOREIGN KEY (
            transaction_id,
            wallet_id,
            currency,
            amount_minor,
            transaction_status
        )
        REFERENCES wager_transactions (
            id,
            wallet_id,
            currency,
            amount_minor,
            status
        ),

    CONSTRAINT ledger_transaction_status_check
        CHECK (transaction_status = 'PROCESSED'),

    CONSTRAINT ledger_direction_check
        CHECK (direction IN ('DEBIT', 'CREDIT')),

    CONSTRAINT ledger_amount_check
        CHECK (amount_minor > 0),

    CONSTRAINT ledger_balances_check
        CHECK (
            balance_before_minor >= 0
            AND balance_after_minor >= 0
        ),

    CONSTRAINT ledger_balance_math_check
        CHECK (
            (
                direction = 'CREDIT'
                AND balance_after_minor::NUMERIC =
                    balance_before_minor::NUMERIC + amount_minor::NUMERIC
            )
            OR
            (
                direction = 'DEBIT'
                AND balance_after_minor::NUMERIC =
                    balance_before_minor::NUMERIC - amount_minor::NUMERIC
            )
        ),

    CONSTRAINT ledger_wallet_transaction_unique
        UNIQUE (wallet_id, transaction_id)
);

CREATE INDEX ledger_wallet_sequence
    ON wallet_ledger_entries (wallet_id, sequence);

CREATE FUNCTION prevent_ledger_mutation()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'wallet ledger entries are append-only'
        USING ERRCODE = '23001';
END;
$$;

CREATE TRIGGER ledger_prevent_update_delete
    BEFORE UPDATE OR DELETE ON wallet_ledger_entries
    FOR EACH ROW
    EXECUTE FUNCTION prevent_ledger_mutation();

CREATE TRIGGER ledger_prevent_truncate
    BEFORE TRUNCATE ON wallet_ledger_entries
    FOR EACH STATEMENT
    EXECUTE FUNCTION prevent_ledger_mutation();

GRANT SELECT, INSERT ON wallet_ledger_entries TO wagering_app;

GRANT USAGE, SELECT ON SEQUENCE wallet_ledger_entries_sequence_seq
    TO wagering_app;
