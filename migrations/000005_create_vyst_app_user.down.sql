-- Revoke permissions from vyst_app user
REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA public FROM vyst_app;
REVOKE ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public FROM vyst_app;
REVOKE ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA public FROM vyst_app;
REVOKE USAGE ON SCHEMA public FROM vyst_app;
DO $$
BEGIN
    EXECUTE format('REVOKE CONNECT ON DATABASE %I FROM vyst_app', current_database());
END
$$;

-- Drop the user
DROP USER IF EXISTS vyst_app;
