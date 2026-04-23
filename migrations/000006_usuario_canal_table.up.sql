
CREATE TABLE IF NOT EXISTS Usuario_Canal (
    fk_id_canal VARCHAR(512) NOT NULL REFERENCES Canal(id_canal),
    fk_id_usuario VARCHAR(512) NOT NULL REFERENCES Usuario(id_usuario),
    fk_id_evento VARCHAR(512) NOT NULL REFERENCES Evento(id_evento), -- evento que causou esse estado
    membresia VARCHAR(50) NOT NULL, -- join, leave, invite, ban
    PRIMARY KEY (fk_id_canal, fk_id_usuario)
);

CREATE INDEX IF NOT EXISTS idx_usuario_membresia_canal ON Usuario_Canal(fk_id_usuario, membresia);
