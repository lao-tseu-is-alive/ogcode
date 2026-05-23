-- +goose Up
CREATE TABLE IF NOT EXISTS callgraph_agent_config (
    id      INTEGER PRIMARY KEY CHECK (id = 1),
    enabled INTEGER NOT NULL DEFAULT 1
);

-- +goose Down
DROP TABLE IF EXISTS callgraph_agent_config;
