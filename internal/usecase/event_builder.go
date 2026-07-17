package usecase

import (
	"encoding/json"
	"time"

	"github.com/caio-bernardo/dragonite/internal/domain"
	"github.com/caio-bernardo/dragonite/internal/domain/types"
)

func newBaseEvent(canalID string, sender string, tipo string, stateKey *string, content any) *domain.Evento {
	bytes, _ := json.Marshal(content)

	return &domain.Evento{
		CanalID:          canalID,
		Sender:           sender,
		Tipo:             tipo,
		Content:          bytes,
		StateKey:         stateKey,
		OrigemServidorTS: time.Now().UnixMilli(),
	}
}

// cria um evento m.room.create
func buildCreateEvent(canalID string, creatorID string, version string) *domain.Evento {
	content := map[string]string{
		"creator":      creatorID,
		"room_version": version,
	}
	return newBaseEvent(canalID, creatorID, string(types.Create), new(""), content)
}

func buildJoinEvent(canalID string, sender string) *domain.Evento {
	content := map[string]string{
		"membership": "join",
	}
	return newBaseEvent(canalID, sender, string(types.Member), &sender, content)
}

func buildLeaveEvent(canalID string, sender string) *domain.Evento {
	content := map[string]string{
		"membership": "leave",
	}
	return newBaseEvent(canalID, sender, string(types.Member), &sender, content)
}

func buildPowerLevelEvent(canalID string, creatorID string) *domain.Evento {
	content := map[string]any{
		"users_default":  0,
		"events_default": 0,
		"state_default":  50,
		"ban":            50,
		"kick":           50,
		"redact":         50,
		"invite":         0,
		"users": map[string]int{
			creatorID: 100, // creator has maximum permissions
		},
		"events": map[string]int{
			"m.room.name":         50,  // Only moderators (50) or higher can change the name
			"m.room.power_levels": 100, // Only the admin can change power levels
		},
	}
	return newBaseEvent(canalID, creatorID, string(types.PowerLevels), new(""), content)
}

func buildJoinRulesEvent(canalID, creatorID, rule string) *domain.Evento {
	content := map[string]string{
		"join_rule": rule, // Singular
	}
	return newBaseEvent(canalID, creatorID, string(types.JoinRules), new(""), content)
}

func buildNameEvent(canalID, sender, name string) *domain.Evento {
	content := map[string]string{
		"name": name,
	}
	return newBaseEvent(canalID, sender, string(types.Name), new(""), content)
}

func buildTopicEvent(canalID, sender, topic string) *domain.Evento {
	content := map[string]string{
		"topic": topic,
	}
	return newBaseEvent(canalID, sender, string(types.Topic), new(""), content)
}

func buildInviteEvent(canalID, sender, invitee string) *domain.Evento {
	content := map[string]string{
		"membership": "invite",
	}
	return newBaseEvent(canalID, sender, string(types.Member), &invitee, content)
}

func buildAliasEvent(canalID, sender, alias string) *domain.Evento {
	content := map[string]string{
		"alias": alias,
	}
	return newBaseEvent(canalID, sender, string(types.CanonicalAlias), new(""), content)
}

// isProtectedGenesisEvent prevents clients from overriding the foundational DAG.
func isProtectedGenesisEvent(eventType string) bool {
	switch eventType {
	case "m.room.create",
		"m.room.member",
		"m.room.power_levels",
		"m.room.join_rules",
		"m.room.history_visibility",
		"m.room.name",  // Handled explicitly by the Name parameter
		"m.room.topic", // Handled explicitly by the Topic parameter
		"m.room.canonical_alias":
		return true
	default:
		return false
	}
}
func buildInitialStateEvents(roomID, creatorID string, rawEvents []StateEventParams) []*domain.Evento {
	var domainEvents []*domain.Evento

	for _, reqEv := range rawEvents {
		// 1. Block malicious or conflicting state
		if isProtectedGenesisEvent(reqEv.Type) {
			continue // Skip it! We handle these ourselves.
		}

		// 2. Safely resolve the State Key (defaults to empty string if not provided)
		stateKey := ""
		if reqEv.StateKey != nil {
			stateKey = *reqEv.StateKey
		}

		// 3. Use the base builder we created earlier
		newEvent := newBaseEvent(
			roomID,
			creatorID, // The creator is ALWAYS the sender of initial state
			reqEv.Type,
			&stateKey,
			reqEv.Content,
		)

		domainEvents = append(domainEvents, newEvent)
	}

	return domainEvents
}
