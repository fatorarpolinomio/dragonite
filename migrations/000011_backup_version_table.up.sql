CREATE TABLE IF NOT EXISTS VersaoBackup (
    id_versao BIGSERIAL PRIMARY KEY,
    id_usuario VARCHAR(512) NOT NULL REFERENCES Usuario(id_usuario) ON DELETE CASCADE,
    algorithm VARCHAR(255) NOT NULL,
    auth_data JSONB NOT NULL,
    count INTEGER NOT NULL DEFAULT 0,
    etag VARCHAR(255) NOT NULL DEFAULT '0',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_versaobackup_usuario ON VersaoBackup (id_usuario, id_versao DESC);