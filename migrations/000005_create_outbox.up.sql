CREATE TABLE outbox_events (
    event_id UUID PRIMARY KEY,
    sequence BIGINT GENERATED ALWAYS AS IDENTITY UNIQUE,
    aggregate_id UUID NOT NULL REFERENCES wallets (id),
    event_type TEXT NOT NULL,
    event_version INTEGER NOT NULL,
    correlation_id TEXT NOT NULL,
    payload JSONB NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,

    attempts INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    locked_by TEXT,
    locked_until TIMESTAMPTZ,
    claim_id UUID,
    last_error TEXT,
    published_at TIMESTAMPTZ,

    CONSTRAINT outbox_event_type_check
        CHECK (
            event_type IN (
                'WagerTransactionProcessed',
                'WagerTransactionRejected',
                'WalletBalanceChanged',
                'WagerTransactionPendingReference'
            )
        ),

    CONSTRAINT outbox_version_check
        CHECK (event_version >= 1),

    CONSTRAINT outbox_correlation_check
        CHECK (btrim(correlation_id) <> ''),

    CONSTRAINT outbox_attempts_check
        CHECK (attempts >= 0),

    CONSTRAINT outbox_payload_object_check
        CHECK (jsonb_typeof(payload) = 'object'),

    CONSTRAINT outbox_payload_fields_check
        CHECK (
            payload ?& ARRAY[
                'eventId',
                'eventType',
                'aggregateId',
                'correlationId',
                'occurredAt',
                'version',
                'data'
            ]
        ),

    CONSTRAINT outbox_payload_types_check
        CHECK (
            jsonb_typeof(payload -> 'eventId') = 'string'
            AND jsonb_typeof(payload -> 'eventType') = 'string'
            AND jsonb_typeof(payload -> 'aggregateId') = 'string'
            AND jsonb_typeof(payload -> 'correlationId') = 'string'
            AND jsonb_typeof(payload -> 'occurredAt') = 'string'
            AND jsonb_typeof(payload -> 'version') = 'number'
            AND jsonb_typeof(payload -> 'data') = 'object'
        ),

    CONSTRAINT outbox_payload_metadata_check
        CHECK (
            payload ->> 'eventId' = event_id::TEXT
            AND payload ->> 'eventType' = event_type
            AND payload ->> 'aggregateId' = aggregate_id::TEXT
            AND payload ->> 'correlationId' = correlation_id
            AND (payload ->> 'version')::NUMERIC = event_version
            AND (payload ->> 'occurredAt')::TIMESTAMPTZ = occurred_at
        ),

    CONSTRAINT outbox_claim_check
        CHECK (
            (
                locked_by IS NULL
                AND locked_until IS NULL
                AND claim_id IS NULL
            )
            OR
            (
                locked_by IS NOT NULL
                AND btrim(locked_by) <> ''
                AND locked_until IS NOT NULL
                AND claim_id IS NOT NULL
            )
        ),

    CONSTRAINT outbox_published_check
        CHECK (
            published_at IS NULL
            OR (
                attempts > 0
                AND locked_by IS NULL
                AND locked_until IS NULL
                AND claim_id IS NULL
            )
        )
);

CREATE INDEX outbox_pending_due
    ON outbox_events (next_attempt_at, sequence)
    WHERE published_at IS NULL;

CREATE INDEX outbox_aggregate_pending
    ON outbox_events (aggregate_id, sequence)
    WHERE published_at IS NULL;

CREATE FUNCTION protect_outbox_event()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF ROW(
        NEW.event_id,
        NEW.sequence,
        NEW.aggregate_id,
        NEW.event_type,
        NEW.event_version,
        NEW.correlation_id,
        NEW.payload,
        NEW.occurred_at
    ) IS DISTINCT FROM ROW(
        OLD.event_id,
        OLD.sequence,
        OLD.aggregate_id,
        OLD.event_type,
        OLD.event_version,
        OLD.correlation_id,
        OLD.payload,
        OLD.occurred_at
    ) THEN
        RAISE EXCEPTION 'outbox event snapshot is immutable'
            USING ERRCODE = '23001';
    END IF;

    IF OLD.published_at IS NOT NULL AND NEW IS DISTINCT FROM OLD THEN
        RAISE EXCEPTION 'published outbox event is immutable'
            USING ERRCODE = '23001';
    END IF;

    RETURN NEW;
END;
$$;

CREATE TRIGGER outbox_protect_snapshot
    BEFORE UPDATE ON outbox_events
    FOR EACH ROW
    EXECUTE FUNCTION protect_outbox_event();

GRANT SELECT, INSERT ON outbox_events TO wagering_app;

GRANT UPDATE (
    attempts,
    next_attempt_at,
    locked_by,
    locked_until,
    claim_id,
    last_error,
    published_at
) ON outbox_events TO wagering_app;

GRANT USAGE, SELECT ON SEQUENCE outbox_events_sequence_seq
    TO wagering_app;
