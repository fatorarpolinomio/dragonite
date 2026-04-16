
-- Representa o estado atual do canal
CREATE TABLE IF NOT EXISTS Estado_Atual_Canal (
    fk_id_canal VARCHAR(512) NOT NULL REFERENCES Canal(id_canal),
    tipo VARCHAR(255) NOT NULL,
    state_key VARCHAR(255) NOT NULL,
    fk_id_evento VARCHAR(512) NOT NULL REFERENCES Evento(id_evento),
    PRIMARY KEY (fk_id_canal, tipo, state_key)
);
