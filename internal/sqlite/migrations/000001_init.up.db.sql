CREATE TABLE IF NOT EXISTS chats (
    id VARCHAR PRIMARY KEY NOT NULL,
    name VARCHAR NOT NULL,
    color VARCHAR NOT NULL DEFAULT '#a187d5',
    type INT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at_clock_value INT NOT NULL DEFAULT 0,
    public_key BLOB,
    unviewed_message_count INT NOT NULL DEFAULT 0,
    last_clock_value INT NOT NULL DEFAULT 0,
    last_message_content_type VARCHAR,
    last_message_content VARCHAR
) WITHOUT ROWID;
