-- +goose Up
CREATE TABLE IF NOT EXISTS call_node (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    directory   TEXT NOT NULL,
    package     TEXT NOT NULL,
    symbol      TEXT NOT NULL,
    file_path   TEXT NOT NULL DEFAULT '',
    line        INTEGER NOT NULL DEFAULT 0,
    kind        TEXT NOT NULL DEFAULT 'function',
    signature   TEXT NOT NULL DEFAULT '',
    doc         TEXT NOT NULL DEFAULT '',
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL,
    UNIQUE(directory, package, symbol)
);

CREATE INDEX IF NOT EXISTS idx_call_node_directory ON call_node(directory);
CREATE INDEX IF NOT EXISTS idx_call_node_package ON call_node(directory, package);
CREATE INDEX IF NOT EXISTS idx_call_node_kind ON call_node(directory, kind);

CREATE TABLE IF NOT EXISTS call_edge (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    directory   TEXT NOT NULL,
    caller_id  INTEGER NOT NULL REFERENCES call_node(id) ON DELETE CASCADE,
    callee_id  INTEGER NOT NULL REFERENCES call_node(id) ON DELETE CASCADE,
    call_type   TEXT NOT NULL DEFAULT 'direct',
    created_at  INTEGER NOT NULL,
    UNIQUE(directory, caller_id, callee_id, call_type)
);

CREATE INDEX IF NOT EXISTS idx_call_edge_directory ON call_edge(directory);
CREATE INDEX IF NOT EXISTS idx_call_edge_caller ON call_edge(caller_id);
CREATE INDEX IF NOT EXISTS idx_call_edge_callee ON call_edge(callee_id);

-- +goose Down
DROP TABLE IF EXISTS call_edge;
DROP TABLE IF EXISTS call_node;