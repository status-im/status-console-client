CREATE TABLE IF NOT EXISTS user_messages (
id VARCHAR PRIMARY KEY NOT NULL,
contact_id VARCHAR NOT NULL,
content_type VARCHAR,
message_type VARCHAR,
text TEXT,
clock BIGINT,
timestamp BIGINT,
content_chat_id TEXT,
content_text TEXT,
public_key BLOB
) WITHOUT ROWID;
CREATE TABLE IF NOT EXISTS user_contacts (
id VARCHAR PRIMARY KEY NOT NULL,
name VARCHAR NOT NULL,
type INT NOT NULL,
public_key BLOB
) WITHOUT ROWID;
