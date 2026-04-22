ALTER TABLE canal
    DROP COLUMN IF EXISTS local_part,
    DROP COLUMN IF EXISTS server_name,
    DROP COLUMN IF EXISTS canonical_alias,
    DROP COLUMN IF EXISTS join_rules,
    DROP COLUMN IF EXISTS guest_access,
    DROP COLUMN IF EXISTS member_count,
    DROP COLUMN IF EXISTS room_type,
    DROP COLUMN IF EXISTS history_visibility;

ALTER TABLE usuario_canal
    DROP COLUMN IF EXISTS joined_at;

ALTER TABLE usuario_canal
    ALTER COLUMN fk_id_evento SET NOT NULL;