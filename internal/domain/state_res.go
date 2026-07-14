package domain

// Identifica estados unicos dentro do canal
type StateTuple struct {
	EventType string
	StateKey  string
}

func NewStateTuple(eventType string, stateKey *string) StateTuple {
	sk := ""
	if stateKey != nil {
		sk = *stateKey
	}
	return StateTuple{
		EventType: eventType,
		StateKey:  sk,
	}
}

// Mapeia uma tupla ao evento vencedor do conflito
type StateMap map[StateTuple]string

// Dados necessários para o State Resol V2
type StateResolutionInput struct {
	RoomID        string
	StateSets     []StateMap
	AuthEventsMap map[string]*Evento
	EventsMap     map[string]*Evento
}
