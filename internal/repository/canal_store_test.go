package repository

import (
	"context"
	"errors"
	"testing"
	"time"
	

	"github.com/caio-bernardo/dragonite/internal/model"
	"github.com/caio-bernardo/dragonite/internal/types"
	"github.com/caio-bernardo/dragonite/internal/util"
)

func TestCanalStoreCRUDAndCleanup(t *testing.T) {
	resetTables(t)

	owner := model.Usuario{
		ID:          "@channel-owner:example.com",
		LocalPart:   "channel-owner",
		Nome:        "Channel Owner",
		Senha:       "password",
		Foto:        "https://example.com/channel-owner.png",
		DataCriacao: baseTime,
	}
	insertUsuario(t, owner)

	store := NewChannelStore(testDB)
	ctx := context.Background()

	canal := model.Canal{
		ID:          "!room:example.com",
		Nome:        "General",
		Descricao:   "General discussion",
		Foto:        "https://example.com/room.png",
		IsPublic:    true,
		Versao:      "1",
		CriadorID:   owner.ID,
		DataCriacao: baseTime.Add(2 * time.Hour),
	}

	if err := store.Create(ctx, &canal); err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	got, err := store.GetByID(ctx, canal.ID)
	if err != nil {
		t.Fatalf("GetByID() failed: %v", err)
	}
	if got.ID != canal.ID || got.Nome != canal.Nome || got.IsPublic != canal.IsPublic || got.CriadorID != canal.CriadorID {
		t.Fatalf("GetByID() returned unexpected canal: %#v", got)
	}

	all, err := store.GetAll(ctx, util.Filter{})
	if err != nil {
		t.Fatalf("GetAll() failed: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("GetAll() expected 1 canal, got %d", len(all))
	}

	updated := canal
	updated.Nome = "General Updated"
	updated.Descricao = "Updated description"
	updated.IsPublic = false
	updated.Foto = "https://example.com/room-updated.png"

	if err := store.Update(ctx, &updated); err != nil {
		t.Fatalf("Update() failed: %v", err)
	}

	gotUpdated, err := store.GetByID(ctx, canal.ID)
	if err != nil {
		t.Fatalf("GetByID() after update failed: %v", err)
	}
	if gotUpdated.Nome != updated.Nome || gotUpdated.Descricao != updated.Descricao || gotUpdated.IsPublic != updated.IsPublic {
		t.Fatalf("Update() did not persist changes: %#v", gotUpdated)
	}

	evento := model.Evento{
		ID:               "$event-1:example.com",
		Tipo:             "m.room.message",
		CanalID:          canal.ID,
		SenderID:         owner.ID,
		StateKey:         "",
		Conteudo:         `{"body":"hello"}`,
		OrigemServidorTS: 1234567890,
		StreamOrdering:   1,
	}
	insertEvento(t, evento)

	insertUsuarioCanal(t, model.UsuarioCanal{
		CanalID:   canal.ID,
		UsuarioID: owner.ID,
		EventoID:  &evento.ID,
		Membresia: "join",
	})

	insertEstadoAtualCanal(t, canal.ID, "m.room.member", owner.ID, evento.ID)

	deleted, err := store.Delete(ctx, canal.ID)
	if err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}
	if deleted.ID != canal.ID {
		t.Fatalf("Delete() returned unexpected canal: %#v", deleted)
	}

	if _, err := store.GetByID(ctx, canal.ID); !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got: %v", err)
	}
}


func TestCanalStore_ListPublic(t *testing.T) {
	resetTables(t)

	owner := model.Usuario{
		ID: "@list-owner:example.com", LocalPart: "list-owner",
		Nome: "Owner", Senha: "password", DataCriacao: baseTime,
	}
	insertUsuario(t, owner)

	store := NewChannelStore(testDB)
	ctx := context.Background()

	c1 := model.Canal{
		ID: "!room1:example.com", LocalPart: "room1", ServerName: "example.com",
		Nome: "Room 1", IsPublic: true, JoinRules: "public", GuestAccess: "forbidden",
		HistoryVisibility: "shared", Versao: "11", CriadorID: owner.ID,
		MemberCount: 1, DataCriacao: baseTime,
	}
	c2 := model.Canal{
		ID: "!room2:example.com", LocalPart: "room2", ServerName: "example.com",
		Nome: "Room 2", IsPublic: true, JoinRules: "public", GuestAccess: "forbidden",
		HistoryVisibility: "shared", Versao: "11", CriadorID: owner.ID,
		MemberCount: 5, DataCriacao: baseTime,
	}

	for _, c := range []model.Canal{c1, c2} {
		if err := store.Create(ctx, &c); err != nil {
			t.Fatalf("Create() failed: %v", err)
		}
	}

	canais, nextBatch, err := store.ListPublic(ctx, 10, "")
	if err != nil {
		t.Fatalf("ListPublic() failed: %v", err)
	}
	if len(canais) != 2 {
		t.Fatalf("expected 2 canais, got %d", len(canais))
	}
	// Ordenado por member_count DESC
	if canais[0].ID != c2.ID {
		t.Fatalf("expected canal with more members first")
	}
	if nextBatch != "" {
		t.Fatalf("expected empty nextBatch, got %q", nextBatch)
	}
}

func TestCanalStore_GetByID_NotFound(t *testing.T) {
	resetTables(t)

	store := NewChannelStore(testDB)
	_, err := store.GetByID(context.Background(), "!naoexiste:example.com")
	if !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

func TestCanalStore_UpdateMemberCount(t *testing.T) {
	resetTables(t)

	owner := model.Usuario{
		ID: "@count-owner:example.com", LocalPart: "count-owner",
		Nome: "Owner", Senha: "password", DataCriacao: baseTime,
	}
	insertUsuario(t, owner)

	store := NewChannelStore(testDB)
	ctx := context.Background()

	canal := model.Canal{
		ID: "!count-room:example.com", LocalPart: "count-room", ServerName: "example.com",
		Nome: "Count Room", IsPublic: true, JoinRules: "public", GuestAccess: "forbidden",
		HistoryVisibility: "shared", Versao: "11", CriadorID: owner.ID,
		MemberCount: 1, DataCriacao: baseTime,
	}
	if err := store.Create(ctx, &canal); err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	if err := store.UpdateMemberCount(ctx, canal.ID, +1); err != nil {
		t.Fatalf("UpdateMemberCount(+1) failed: %v", err)
	}

	got, _ := store.GetByID(ctx, canal.ID)
	if got.MemberCount != 2 {
		t.Fatalf("expected MemberCount 2, got %d", got.MemberCount)
	}
}