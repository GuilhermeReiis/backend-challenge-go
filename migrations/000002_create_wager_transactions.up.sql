CREATE TABLE wager_transactions (
                                    id UUID PRIMARY KEY,
                                    origin VARCHAR(8) NOT NULL,
                                    kind VARCHAR(8) NOT NULL,
                                    status VARCHAR(20) NOT NULL,

                                    wallet_id UUID NOT NULL,
                                    player_id UUID NOT NULL,
                                    currency VARCHAR(3) NOT NULL,
                                    amount_minor BIGINT NOT NULL,

                                    provider_id TEXT,
                                    external_transaction_id TEXT,
                                    idempotency_key TEXT,
                                    payload_hash TEXT,
                                    round_id TEXT,
                                    game_id TEXT,

                                    reference_external_transaction_id TEXT,
                                    reference_transaction_id UUID,

                                    failure_code TEXT,
                                    result_balance_minor BIGINT,
                                    processed_at TIMESTAMPTZ,

                                    attempts INTEGER NOT NULL DEFAULT 0,
                                    next_attempt_at TIMESTAMPTZ,
                                    expires_at TIMESTAMPTZ,

                                    created_at TIMESTAMPTZ NOT NULL,
                                    updated_at TIMESTAMPTZ NOT NULL,

                                    CONSTRAINT wager_wallet_currency_fk
                                        FOREIGN KEY (wallet_id, currency)
                                            REFERENCES wallets (id, currency),

                                    CONSTRAINT wager_reference_fk
                                        FOREIGN KEY (reference_transaction_id)
                                            REFERENCES wager_transactions (id),

                                    CONSTRAINT wager_origin_check
                                        CHECK (origin IN ('INTERNAL', 'EXTERNAL')),

                                    CONSTRAINT wager_kind_check
                                        CHECK (kind IN ('OPENING', 'BET', 'WIN', 'LOSS', 'REFUND', 'ROLLBACK')),

                                    CONSTRAINT wager_status_check
                                        CHECK (
                                            status IN (
                                                       'PENDING',
                                                       'PENDING_REFERENCE',
                                                       'PROCESSED',
                                                       'REJECTED',
                                                       'FAILED'
                                                )
                                            ),

                                    CONSTRAINT wager_amount_check
                                        CHECK (
                                            (kind = 'LOSS' AND amount_minor = 0)
                                                OR
                                            (kind <> 'LOSS' AND amount_minor > 0)
                                            ),

                                    CONSTRAINT wager_origin_kind_check
                                        CHECK (
                                            (origin = 'INTERNAL' AND kind = 'OPENING')
                                                OR
                                            (origin = 'EXTERNAL' AND kind <> 'OPENING')
                                            ),

                                    CONSTRAINT wager_external_metadata_check
                                        CHECK (
                                            origin <> 'EXTERNAL'
                                                OR (
                                                provider_id IS NOT NULL AND btrim(provider_id) <> ''
                                                    AND external_transaction_id IS NOT NULL
                                                    AND btrim(external_transaction_id) <> ''
                                                    AND idempotency_key IS NOT NULL AND btrim(idempotency_key) <> ''
                                                    AND payload_hash IS NOT NULL
                                                    AND payload_hash ~ '^[0-9a-f]{64}$'
                AND round_id IS NOT NULL AND btrim(round_id) <> ''
                AND game_id IS NOT NULL AND btrim(game_id) <> ''
                                                )
                                            ),

                                    CONSTRAINT wager_internal_metadata_check
                                        CHECK (
                                            origin <> 'INTERNAL'
                                                OR (
                                                provider_id IS NULL
                                                    AND external_transaction_id IS NULL
                                                    AND idempotency_key IS NULL
                                                    AND payload_hash IS NULL
                                                    AND round_id IS NULL
                                                    AND game_id IS NULL
                                                    AND reference_external_transaction_id IS NULL
                                                    AND reference_transaction_id IS NULL
                                                )
                                            ),

                                    CONSTRAINT wager_reference_shape_check
                                        CHECK (
                                            reference_external_transaction_id IS NULL
                                                OR btrim(reference_external_transaction_id) <> ''
                                            ),

                                    CONSTRAINT wager_reference_kind_check
                                        CHECK (
                                            (
                                                kind IN ('REFUND', 'ROLLBACK')
                                                    AND reference_external_transaction_id IS NOT NULL
                                                )
                                                OR kind = 'WIN'
                                                OR (
                                                kind IN ('OPENING', 'BET', 'LOSS')
                                                    AND reference_external_transaction_id IS NULL
                                                )
                                            ),

                                    CONSTRAINT wager_resolved_reference_check
                                        CHECK (
                                            reference_transaction_id IS NULL
                                                OR (
                                                reference_external_transaction_id IS NOT NULL
                                                    AND reference_transaction_id <> id
                                                )
                                            ),

                                    CONSTRAINT wager_processed_reference_check
                                        CHECK (
                                            status <> 'PROCESSED'
                                                OR reference_external_transaction_id IS NULL
                                                OR reference_transaction_id IS NOT NULL
                                            ),

                                    CONSTRAINT wager_pending_reference_check
                                        CHECK (
                                            status <> 'PENDING_REFERENCE'
                                                OR (
                                                reference_external_transaction_id IS NOT NULL
                                                    AND next_attempt_at IS NOT NULL
                                                    AND expires_at IS NOT NULL
                                                )
                                            ),

                                    CONSTRAINT wager_failure_code_check
                                        CHECK (
                                            (status IN ('REJECTED', 'FAILED')) = (failure_code IS NOT NULL)
                                            ),

                                    CONSTRAINT wager_failure_code_content_check
                                        CHECK (failure_code IS NULL OR btrim(failure_code) <> ''),

                                    CONSTRAINT wager_result_check
                                        CHECK (
                                            (status = 'PROCESSED') = (result_balance_minor IS NOT NULL)
                                            ),

                                    CONSTRAINT wager_result_balance_check
                                        CHECK (result_balance_minor IS NULL OR result_balance_minor >= 0),

                                    CONSTRAINT wager_processed_at_check
                                        CHECK (
                                            (status = 'PROCESSED') = (processed_at IS NOT NULL)
                                            ),

                                    CONSTRAINT wager_timestamps_check
                                        CHECK (
                                            updated_at >= created_at
                                                AND (processed_at IS NULL OR processed_at = updated_at)
                                            ),

                                    CONSTRAINT wager_attempts_check
                                        CHECK (attempts >= 0),

                                    CONSTRAINT wager_expiration_check
                                        CHECK (expires_at IS NULL OR expires_at > created_at),

                                    CONSTRAINT wager_id_wallet_currency_unique
                                        UNIQUE (id, wallet_id, currency)
);

CREATE UNIQUE INDEX wager_provider_external_unique
    ON wager_transactions (provider_id, external_transaction_id)
    WHERE origin = 'EXTERNAL';

CREATE UNIQUE INDEX wager_provider_idempotency_unique
    ON wager_transactions (provider_id, idempotency_key)
    WHERE origin = 'EXTERNAL';

CREATE UNIQUE INDEX wager_opening_wallet_unique
    ON wager_transactions (wallet_id)
    WHERE kind = 'OPENING';

CREATE UNIQUE INDEX wager_successful_reversal_unique
    ON wager_transactions (reference_transaction_id)
    WHERE kind IN ('REFUND', 'ROLLBACK') AND status = 'PROCESSED';

CREATE INDEX wager_pending_reference_due
    ON wager_transactions (next_attempt_at, id)
    WHERE status = 'PENDING_REFERENCE';

CREATE INDEX wager_wallet_created
    ON wager_transactions (wallet_id, created_at, id);

GRANT SELECT, INSERT, UPDATE ON wager_transactions TO wagering_app;