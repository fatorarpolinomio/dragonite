package model

import "time"

type Dispositivo struct {
	ID                    string    `json:"id_dispositivo"`
	Nome                  string    `json:"nome_dispositivo"`
	UsuarioID             string    `json:"usuario_id"`
	RefreshToken          string    `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time `json:"refresh_token_expires_at"`
	UltimoIPVisto         string    `json:"ultimo_ip_visto"`
	UltimoTimestampVisto  time.Time `json:"ultimo_timestamp_visto"`
}

type DispositivoCreate struct {
	UsuarioID             string    `json:"usuario_id"`
	Nome                  string    `json:"nome_dispositivo"`
	RefreshToken          string    `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time `json:"refresh_token_expires_at"`
	UltimoIPVisto         string    `json:"ultimo_ip_visto"`
	UltimoTimestampVisto  time.Time `json:"ultimo_timestamp_visto"`
}

func (dr DispositivoCreate) ToDispositivo() Dispositivo {
	return Dispositivo{
		UsuarioID:             dr.UsuarioID,
		Nome:                  dr.Nome,
		RefreshToken:          dr.RefreshToken,
		RefreshTokenExpiresAt: dr.RefreshTokenExpiresAt,
		UltimoIPVisto:         dr.UltimoIPVisto,
		UltimoTimestampVisto:  dr.UltimoTimestampVisto,
	}
}
