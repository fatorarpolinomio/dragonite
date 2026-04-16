package model

type UsuarioCanal struct {
	CanalID   string `json:"canal_id"`
	UsuarioID string `json:"usuario_id"`
	EventoID  string `json:"evento_id"`
	Membresia string `json:"membresia"`
}
