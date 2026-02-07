\set pgpass `echo "$POSTGRES_PASSWORD"`

CREATE USER datalk_admin WITH PASSWORD :'pgpass' SUPERUSER;
CREATE USER auth_admin WITH PASSWORD :'pgpass' SUPERUSER;

ALTER SCHEMA datalk OWNER TO datalk_admin;
ALTER SCHEMA auth OWNER TO auth_admin;

CREATE TYPE auth.factor_type AS ENUM('totp', 'webauthn');
CREATE TYPE auth.factor_status AS ENUM('unverified', 'verified');
CREATE TYPE auth.aal_level AS ENUM('aal1', 'aal2', 'aal3');
CREATE TYPE auth.code_challenge_method AS ENUM('s256', 'plain');
CREATE TYPE auth.one_time_token_type AS ENUM (
    'confirmation_token',
    'reauthentication_token',
    'recovery_token',
    'email_change_token_new',
    'email_change_token_current',
    'phone_change_token'
);
