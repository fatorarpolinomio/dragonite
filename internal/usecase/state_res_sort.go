package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"sort"

	"github.com/caio-bernardo/dragonite/internal/domain"
)

// Resolve conflitos de estado dentro do DAG de um canal
type StateResolverService struct {
	authResolver interface {
		CheckAuth(ctx context.Context, event *domain.Evento, currentState domain.StateMap, eventsMap map[string]*domain.Evento) bool
	}
}

func NewStateResolverService(authResolver interface {
	CheckAuth(ctx context.Context, event *domain.Evento, currentState domain.StateMap, eventsMap map[string]*domain.Evento) bool
}) *StateResolverService {
	return &StateResolverService{
		authResolver: authResolver,
	}
}

// partitionState splits state into Unconflicted (identical across all sets) and Conflicted.
func (s *StateResolverService) partitionState(sets []domain.StateMap) (domain.StateMap, []domain.StateTuple) {
	unconflicted := make(domain.StateMap)
	tupleCounts := make(map[domain.StateTuple]map[string]int)

	for _, set := range sets {
		for tuple, eventID := range set {
			if tupleCounts[tuple] == nil {
				tupleCounts[tuple] = make(map[string]int)
			}
			tupleCounts[tuple][eventID]++
		}
	}

	var conflicted []domain.StateTuple
	for tuple, ids := range tupleCounts {
		if len(ids) == 1 {
			for eventID, count := range ids {
				if count == len(sets) {
					unconflicted[tuple] = eventID
				} else {
					conflicted = append(conflicted, tuple)
				}
			}
		} else {
			conflicted = append(conflicted, tuple)
		}
	}
	return unconflicted, conflicted
}

// Resolve executes the full Matrix State Resolution v2 algorithm.
func (s *StateResolverService) Resolve(ctx context.Context, input domain.StateResolutionInput) (domain.StateMap, error) {
	if len(input.StateSets) == 0 {
		return nil, fmt.Errorf("no state sets provided for resolution")
	}
	if len(input.StateSets) == 1 {
		return input.StateSets[0], nil
	}

	// 1. Partition into unconflicted and conflicted state
	unconflicted, conflictedTuples := s.partitionState(input.StateSets)
	if len(conflictedTuples) == 0 {
		return unconflicted, nil
	}

	// 2. Identify all conflicted Event IDs and their full auth chains
	conflictedEventIDsMap := make(map[string]bool)
	for _, set := range input.StateSets {
		for _, tuple := range conflictedTuples {
			if id, ok := set[tuple]; ok {
				conflictedEventIDsMap[id] = true
			}
		}
	}

	// Build a combined map of all known events for easy lookup
	allEventsMap := make(map[string]*domain.Evento)
	maps.Copy(allEventsMap, input.EventsMap)
	maps.Copy(allEventsMap, input.AuthEventsMap)

	// 3. Extract unconflicted Power Levels event for sorting tie-breakers
	var plEvent *domain.Evento
	plTuple := domain.StateTuple{EventType: "m.room.power_levels", StateKey: ""}
	if plID, ok := unconflicted[plTuple]; ok {
		plEvent = allEventsMap[plID]
	}

	// 4. Group conflicted events into the 3 resolution phases
	var phase1IDs, phase2IDs, phase3IDs []string
	for id := range conflictedEventIDsMap {
		ev, exists := allEventsMap[id]
		if !exists {
			continue
		}
		switch ev.Tipo {
		case "m.room.create", "m.room.power_levels", "m.room.join_rules":
			phase1IDs = append(phase1IDs, id)
		case "m.room.member", "m.room.third_party_invite":
			phase2IDs = append(phase2IDs, id)
		default:
			phase3IDs = append(phase3IDs, id)
		}
	}

	// 5. Initialize the accumulator with the unconflicted baseline
	resolvedState := make(domain.StateMap)
	maps.Copy(resolvedState, unconflicted)

	// Helper function to iteratively sort and apply a phase of events
	applyPhase := func(eventIDs []string) {
		sortedIDs := sortEventsTopologically(eventIDs, allEventsMap, plEvent)
		for _, id := range sortedIDs {
			ev := allEventsMap[id]
			tuple := domain.NewStateTuple(ev.Tipo, ev.StateKey)

			// Validate if this candidate passes auth against the currently resolved state
			if s.authResolver.CheckAuth(ctx, ev, resolvedState, allEventsMap) {
				resolvedState[tuple] = id
				// Update power levels reference immediately if power levels just got resolved
				if tuple.EventType == "m.room.power_levels" && tuple.StateKey == "" {
					plEvent = ev
				}
			}
		}
	}

	// 6. Execute the phases sequentially
	applyPhase(phase1IDs)
	applyPhase(phase2IDs)
	applyPhase(phase3IDs)

	return resolvedState, nil
}

// Extrai o power level do usuário com base no evento m.room.power_levels
func getEffectivePowerLevel(sender string, powerLevelsEvent *domain.Evento) int64 {
	if powerLevelsEvent == nil {
		return 0
	}

	var content struct {
		Users        map[string]int64 `json:"users"`
		UsersDefault int64            `json:"users_default"`
	}
	if err := json.Unmarshal(powerLevelsEvent.Content, &content); err != nil {
		return 0
	}

	if pl, exists := content.Users[sender]; exists {
		return pl
	}
	return content.UsersDefault
}

// Ordenação de eventos com base no Algoritmo de Kahn e regras do State Res v2 do Matrix (simplificado)
func sortEventsTopologically(eventIDs []string, eventsMap map[string]*domain.Evento, plEvent *domain.Evento) []string {
	inDegree := make(map[string]int)
	graph := make(map[string][]string)

	// Initialize degrees
	for _, id := range eventIDs {
		inDegree[id] = 0
	}

	// Build dependency graph based on auth_events
	for _, id := range eventIDs {
		ev, exists := eventsMap[id]
		if !exists {
			continue
		}
		for _, authID := range ev.AuthEventos {
			if _, inGraph := inDegree[authID]; inGraph {
				graph[authID] = append(graph[authID], id)
				inDegree[id]++
			}
		}
	}

	// Find initial nodes with 0 in-degree
	var zeroDegree []string
	for id, deg := range inDegree {
		if deg == 0 {
			zeroDegree = append(zeroDegree, id)
		}
	}

	// Comparator for tie-breaking
	sortCandidates := func(candidates []string) {
		sort.SliceStable(candidates, func(i, j int) bool {
			evI := eventsMap[candidates[i]]
			evJ := eventsMap[candidates[j]]
			if evI == nil || evJ == nil {
				return candidates[i] < candidates[j]
			}

			plI := getEffectivePowerLevel(evI.Sender, plEvent)
			plJ := getEffectivePowerLevel(evJ.Sender, plEvent)

			// 1. Higher Power Level comes first
			if plI != plJ {
				return plI > plJ
			}

			// 2. Earlier Timestamp comes first
			if evI.OrigemServidorTS != evJ.OrigemServidorTS {
				return evI.OrigemServidorTS < evJ.OrigemServidorTS
			}

			// 3. Lexicographical comparison of Event ID
			return evI.ID < evJ.ID
		})
	}

	var sorted []string
	for len(zeroDegree) > 0 {
		sortCandidates(zeroDegree)

		// Pop the winning event
		curr := zeroDegree[0]
		zeroDegree = zeroDegree[1:]
		sorted = append(sorted, curr)

		// Reduce in-degree for dependent events
		for _, neighbor := range graph[curr] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				zeroDegree = append(zeroDegree, neighbor)
			}
		}
	}

	return sorted
}
