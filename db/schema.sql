DROP TABLE IF EXISTS freelancers;
DROP TABLE IF EXISTS clients;
DROP TABLE IF EXISTS emails;
DROP TYPE IF EXISTS user_role;

CREATE TYPE user_role AS ENUM ('client', 'freelancer');

CREATE TABLE emails (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    role user_role NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE clients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) NOT NULL UNIQUE,
    email_id UUID NOT NULL UNIQUE REFERENCES emails(id),
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE freelancers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) NOT NULL UNIQUE,
    email_id UUID NOT NULL UNIQUE REFERENCES emails(id),
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);