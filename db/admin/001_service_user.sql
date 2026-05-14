\if :{?app_password}
\else
\echo 'ERROR: psql variable app_password must be provided'
\quit 1
\endif

SELECT format(
    'CREATE ROLE asap_app LOGIN PASSWORD %L NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION',
    :'app_password'
)
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'asap_app')
\gexec

SELECT format('ALTER ROLE asap_app PASSWORD %L', :'app_password')
\gexec

GRANT CONNECT ON DATABASE asap TO asap_app;
GRANT USAGE ON SCHEMA public TO asap_app;

GRANT SELECT, INSERT, UPDATE, DELETE
ON ALL TABLES IN SCHEMA public
TO asap_app;

GRANT USAGE, SELECT
ON ALL SEQUENCES IN SCHEMA public
TO asap_app;

ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA public
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO asap_app;

ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA public
GRANT USAGE, SELECT ON SEQUENCES TO asap_app;
