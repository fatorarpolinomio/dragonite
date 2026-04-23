CREATE TABLE IF NOT EXISTS Canal (
    id_canal varchar(512) PRIMARY KEY,
    nome_canal varchar(50) NOT NULL,
    descricao_canal text,
    foto_canal text,
    is_public_canal boolean NOT NULL DEFAULT false,
    versao_canal VARCHAR(50) DEFAULT '1', -- versão da sala (usado pelo state resolution)
    fk_id_criador varchar(512) NOT NULL REFERENCES usuario(id_usuario),
    data_criacao_canal TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
