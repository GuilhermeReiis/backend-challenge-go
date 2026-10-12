BEGIN;

DROP TRIGGER ledger_validate_operation ON wallet_ledger_entries;
DROP FUNCTION validate_ledger_operation();
DROP FUNCTION ledger_expected_direction(TEXT, TEXT);

ALTER TABLE wager_transactions
    DROP CONSTRAINT wager_wallet_player_currency_fk;

ALTER TABLE wallets
    DROP CONSTRAINT wallets_id_player_currency_unique;

COMMIT;
