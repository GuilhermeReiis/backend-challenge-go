\set ON_ERROR_STOP on
\getenv app_password APP_DB_PASSWORD

CREATE ROLE wagering_app
    WITH LOGIN PASSWORD :'app_password';

GRANT USAGE ON SCHEMA public TO wagering_app;