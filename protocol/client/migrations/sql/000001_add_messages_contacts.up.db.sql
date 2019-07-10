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

CREATE TABLE IF NOT EXISTS user_messages (
id BLOB UNIQUE NOT NULL,
chat_id VARCHAR NOT NULL,
content_type VARCHAR,
message_type VARCHAR,
text TEXT,
clock BIGINT,
timestamp BIGINT,
content_chat_id TEXT,
content_text TEXT,
public_key BLOB,
flags INT NOT NULL DEFAULT 0
);

CREATE INDEX chat_ids ON user_messages(chat_id);

CREATE TABLE IF NOT EXISTS membership_updates (
  id VARCHAR PRIMARY KEY NOT NULL,
  data BLOB NOT NULL,
  chat_id VARCHAR NOT NULL,
  FOREIGN KEY (chat_id) REFERENCES chats(id)
  ) WITHOUT ROWID;

  CREATE TABLE IF NOT EXISTS chat_members (
    public_key BLOB NOT NULL,
    chat_id VARCHAR NOT NULL,
    admin BOOLEAN NOT NULL DEFAULT FALSE,
    joined BOOLEAN NOT NULL DEFAULT FALSE,
    FOREIGN KEY (chat_id) REFERENCES chats(id),
    UNIQUE(chat_id, public_key));
