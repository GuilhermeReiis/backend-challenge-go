CREATE TABLE inbox_messages (
    consumer_name TEXT NOT NULL,
    message_id TEXT NOT NULL,
    payload_hash VARCHAR(64) NOT NULL,
    received_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,

    CONSTRAINT inbox_primary_key
        PRIMARY KEY (consumer_name, message_id),

    CONSTRAINT inbox_consumer_name_check
        CHECK (btrim(consumer_name) <> ''),

    CONSTRAINT inbox_message_id_check
        CHECK (btrim(message_id) <> ''),

    CONSTRAINT inbox_payload_hash_check
        CHECK (payload_hash ~ '^[0-9a-f]{64}$'),

    CONSTRAINT inbox_timestamps_check
        CHECK (
            completed_at IS NULL
            OR completed_at >= received_at
        )
);

GRANT SELECT, INSERT ON inbox_messages TO wagering_app;

GRANT UPDATE (completed_at) ON inbox_messages TO wagering_app;
