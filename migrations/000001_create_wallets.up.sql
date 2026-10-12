CREATE TABLE wallets (
                         id UUID PRIMARY KEY,
                         player_id UUID NOT NULL,
                         currency VARCHAR(3) NOT NULL,
                         balance_minor BIGINT NOT NULL,
                         version BIGINT NOT NULL DEFAULT 1,
                         created_at TIMESTAMPTZ NOT NULL,
                         updated_at TIMESTAMPTZ NOT NULL,

                         CONSTRAINT wallets_currency_check
                             CHECK (currency IN ('BRL', 'USD')),

                         CONSTRAINT wallets_balance_check
                             CHECK (balance_minor >= 0),

                         CONSTRAINT wallets_version_check
                             CHECK (version >= 1),

                         CONSTRAINT wallets_timestamps_check
                             CHECK (updated_at >= created_at),

                         CONSTRAINT wallets_player_currency_unique
                             UNIQUE (player_id, currency),

                         CONSTRAINT wallets_id_currency_unique
                             UNIQUE (id, currency)
);

GRANT SELECT, INSERT, UPDATE ON wallets TO wagering_app;