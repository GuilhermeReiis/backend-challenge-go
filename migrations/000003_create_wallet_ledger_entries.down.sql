DROP TABLE wallet_ledger_entries;

DROP FUNCTION prevent_ledger_mutation();

ALTER TABLE wager_transactions
    DROP CONSTRAINT wager_ledger_target_unique;
