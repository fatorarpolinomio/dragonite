package model

import "time"

type Canal struct {
	ID          string    `json:"id_canal"`
	Nome        string    `json:"nome_canal"`
	Descricao   string    `json:"descricao_canal"`
	Foto        string    `json:"foto_canal"`
	IsPublic    bool      `json:"is_public_canal"`
	Versao      string    `json:"version_canal"`
	CriadorID   string    `json:"id_criador"`
	DataCriacao time.Time `json:"data_criacao"`
}

type CanalCreate struct {
	Nome      string `json:"nome_canal"`
	Descricao string `json:"descricao_canal"`
	Foto      string `json:"foto_canal"`
	IsPublic  bool   `json:"is_public_canal"`
	Versao    string `json:"version_canal"`
	CriadorID string `json:"id_criador"`
}

func (c CanalCreate) ToCanal() Canal {
	return Canal{
		Nome:        c.Nome,
		Descricao:   c.Descricao,
		Foto:        c.Foto,
		IsPublic:    c.IsPublic,
		Versao:      c.Versao,
		CriadorID:   c.CriadorID,
		DataCriacao: time.Now(),
	}
}

type EstadoAtualCanal struct {
	CanalID  string `json:"id_canal"`
	Tipo     string `json:"tipo"`
	StateKey string `json:"state_key"`
	EventoID string `json:"evento_id"`
}
