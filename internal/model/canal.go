package model

import "time"
import "fmt"

type Canal struct {
	ID          string    `json:"id_canal"`
	LocalPart   string    `json:"localpart_canal"`
	ServerName  string    `json:"server_name"`
	Nome        string    `json:"nome_canal"`
	Descricao   string    `json:"descricao_canal"`
	Foto        string    `json:"foto_canal"`
	CanonAlias  *string   `json:"canonical_alias"`
	IsPublic    bool      `json:"is_public_canal"`
	JoinRules   string    `json:"join_rules"`
	GuestAccess string    `json:"guest_access"`
	RoomType    *string   `json:"room_type"`
	Versao      string    `json:"versao_canal"`
	CriadorID   string    `json:"id_criador"`
	MemberCount int       `json:"member_count"`
	HistoryVisibility string  `json:"history_visibility"`
	DataCriacao time.Time `json:"data_criacao_canal"`
}

type CanalCreate struct {
	LocalPart         string  `json:"local_part"`
	ServerName        string  `json:"server_name"`
	Nome              string  `json:"nome_canal"`
	Descricao         string  `json:"descricao_canal"`
	Foto              string  `json:"foto_canal"`
	IsPublic          bool    `json:"is_public_canal"`
	JoinRules         string  `json:"join_rules"`
	GuestAccess       string  `json:"guest_access"`
	HistoryVisibility string  `json:"history_visibility"`
	RoomType          *string `json:"room_type"`
	Versao            string  `json:"versao_canal"`
	CriadorID         string  `json:"id_criador"`
}

func (c CanalCreate) ToCanal() Canal {
	return Canal{
		ID:                GenerateCanalID(c.LocalPart, c.ServerName),
		LocalPart:         c.LocalPart,
		ServerName:        c.ServerName,
		Nome:              c.Nome,
		Descricao:         c.Descricao,
		Foto:              c.Foto,
		IsPublic:          c.IsPublic,
		JoinRules:         c.JoinRules,
		GuestAccess:       c.GuestAccess,
		HistoryVisibility: c.HistoryVisibility,
		RoomType:          c.RoomType,
		Versao:            c.Versao,
		CriadorID:         c.CriadorID,
		DataCriacao:       time.Now(),
		MemberCount:       1,
	}
}

func GenerateCanalID(localpart, serverName string) string {
    return fmt.Sprintf("!%s:%s", localpart, serverName)
}

type EstadoAtualCanal struct {
	CanalID  string `json:"id_canal"`
	Tipo     string `json:"tipo"`
	StateKey string `json:"state_key"`
	EventoID string `json:"evento_id"`
}
