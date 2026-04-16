CREATE TABLE IF NOT EXISTS Dispositivo (
    id_dispositivo UUID PRIMARY KEY DEFAULT uuidv7(),
    fk_id_usuario varchar(255) NOT NULL REFERENCES usuario(id_usuario) ON DELETE CASCADE,
    nome_dispositivo varchar(255),
    refresh_token varchar(64) NOT NULL UNIQUE, -- token de "refresco" do token de acesso
    refresh_token_expires_at TIMESTAMP WITH TIME ZONE,
    ultimo_ip_visto varchar(50),
    ultimo_timestamp_visto TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_dispositivo_refresh_token ON Dispositivo (refresh_token);
