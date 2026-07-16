package roomkeys

import "encoding/json"

// CreateBackupVersionRequest é o corpo da requisição POST /_matrix/client/v3/room_keys/version
type CreateBackupVersionRequest struct {
	Algorithm string          `json:"algorithm"`
	AuthData  json.RawMessage `json:"auth_data"`
}

// CreateBackupVersionResponse é o corpo da resposta 200 de POST /_matrix/client/v3/room_keys/version
type CreateBackupVersionResponse struct {
	Version string `json:"version"`
}

// BackupVersionResponse é o corpo da resposta 200 de GET /_matrix/client/v3/room_keys/version
type BackupVersionResponse struct {
	Algorithm string          `json:"algorithm"`
	AuthData  json.RawMessage `json:"auth_data"`
	Count     int64           `json:"count"`
	ETag      string          `json:"etag"`
	Version   string          `json:"version"`
}