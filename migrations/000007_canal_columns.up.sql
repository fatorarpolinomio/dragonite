-- Adiciona colunas Matrix que faltam na tabela canal
ALTER TABLE canal
    ADD COLUMN IF NOT EXISTS local_part        VARCHAR(255),
    ADD COLUMN IF NOT EXISTS server_name       VARCHAR(255),
    ADD COLUMN IF NOT EXISTS canonical_alias   VARCHAR(512),
    ADD COLUMN IF NOT EXISTS join_rules        VARCHAR(50) NOT NULL DEFAULT 'invite',
    ADD COLUMN IF NOT EXISTS guest_access      VARCHAR(50) NOT NULL DEFAULT 'forbidden',
    ADD COLUMN IF NOT EXISTS member_count      INTEGER     NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS room_type         VARCHAR(255),
    ADD COLUMN IF NOT EXISTS history_visibility VARCHAR(50) NOT NULL DEFAULT 'shared';

-- joined_at para saber quando o usuário entrou na sala
ALTER TABLE usuario_canal
    ADD COLUMN IF NOT EXISTS joined_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP;

-- fk_id_evento era NOT NULL mas join/leave ainda não geram eventos Matrix
ALTER TABLE usuario_canal
    ALTER COLUMN fk_id_evento DROP NOT NULL;