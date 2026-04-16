CREATE DOMAIN usuario_localpart_type AS VARCHAR(255) CHECK (
    VALUE ~ '^[\w\d._=+-]+$'
);

CREATE TABLE IF NOT EXISTS Usuario (
    id_usuario VARCHAR(512) PRIMARY KEY, -- ex: @joao:servidor1.com
    localpart_usuario usuario_localpart_type NOT NULL UNIQUE, -- ex: joao
    senha_usuario VARCHAR(255) NOT NULL,
    nome_usuario VARCHAR(255), -- ex: Joao Silberto
    foto_usuario VARCHAR(255),
    data_criacao_usuario TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
