CREATE TABLE IF NOT EXISTS state (
    id BLOB NOT NULL,
    `group` BLOB NOT NULL,
    peer BLOB NOT NULL,
    send_count BIG_INT,
    send_epoch BIG_INT
);

CREATE UNIQUE INDEX states ON state(id, `group`, peer);
