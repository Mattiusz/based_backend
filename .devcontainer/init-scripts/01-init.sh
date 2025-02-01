#!/bin/bash
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    -- Grant schema creation permissions to the default user
    ALTER USER user CREATEDB;
    ALTER USER user CREATEROLE;
    
    -- Allow user to create schemas in the database
    ALTER DATABASE myservice_db OWNER TO user;
    GRANT ALL PRIVILEGES ON DATABASE myservice_db TO user;
    
    -- Enable schema creation in public schema
    GRANT ALL ON SCHEMA public TO user;
EOSQL