package model

import "fmt"

type SyncToken struct {
	RoomEvents  int64
	Receipts    int64
	AccountData int64
}

func (t SyncToken) Encode() string {
	return fmt.Sprintf("s%d_%d_%d", t.RoomEvents, t.Receipts, t.AccountData)
}

func ParseToken(t string) SyncToken {
	var token SyncToken
	_, err := fmt.Sscanf(t, "s%d_%d_%d", &token.RoomEvents, &token.Receipts, &token.AccountData)
	if err != nil {
		return SyncToken{
			RoomEvents:  0,
			Receipts:    0,
			AccountData: 0,
		}
	}
	return token
}
