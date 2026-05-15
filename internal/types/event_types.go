package types

// Type is type of event
type EventType string

const (
	// https://matrix.org/docs/spec/client_server/latest#m-room-aliases
	Aliases EventType = "m.room.aliases"

	// https://matrix.org/docs/spec/client_server/latest#m-room-canonical-alias
	CanonicalAlias EventType = "m.room.canonical_alias"

	// https://matrix.org/docs/spec/client_server/latest#m-room-create
	Create EventType = "m.room.create"

	// https://matrix.org/docs/spec/client_server/latest#m-room-join-rules
	JoinRules EventType = "m.room.join_rules"

	// https://matrix.org/docs/spec/client_server/latest#m-room-member
	Member EventType = "m.room.member"

	// https://matrix.org/docs/spec/client_server/latest#m-room-power-levels
	PowerLevels EventType = "m.room.power_levels"

	// https://matrix.org/docs/spec/client_server/latest#m-room-redaction
	Redaction EventType = "m.room.redaction"

	// https://matrix.org/docs/spec/client_server/latest#m-room-message
	Message EventType = "m.room.message"

	// https://matrix.org/docs/spec/client_server/latest#m-room-message-feedback
	MessageFeedback EventType = "m.room.message.feedback"

	// https://matrix.org/docs/spec/client_server/latest#m-room-name
	Name EventType = "m.room.name"

	// https://matrix.org/docs/spec/client_server/latest#m-room-topic
	Topic EventType = "m.room.topic"

	// https://matrix.org/docs/spec/client_server/latest#m-room-avatar
	Avatar EventType = "m.room.avatar"

	// https://matrix.org/docs/spec/client_server/latest#m-room-pinned-events
	PinnedEvents EventType = "m.room.pinned_events"
)
