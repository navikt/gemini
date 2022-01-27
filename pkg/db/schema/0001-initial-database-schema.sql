-- Run the entire migration as an atomic operation.
START TRANSACTION ISOLATION LEVEL SERIALIZABLE READ WRITE;

-- Table apikey holds teams' deploy API keys.
-- A team can have many API keys, with each key having its own expiry time.
CREATE TABLE users
(
    "id"       varchar primary key not null,
    "username" varchar unique      not null,
    "token"    varchar             null
);

-- Database migration
CREATE TABLE migrations
(
    "version" int primary key          not null,
    "created" timestamp with time zone not null
);

-- Mark this database migration as completed.
INSERT INTO migrations (version, created)
VALUES (1, now());
COMMIT;
