-- +goose Up
CREATE TABLE IF NOT EXISTS callgraph_config (
    directory TEXT PRIMARY KEY,
    model     TEXT NOT NULL DEFAULT ''
);

-- +goose Down
DROP TABLE IF EXISTS callgraph_config;
