package usecase

import (
	"context"
	"fmt"
)

// Resolves Authentication Event Dependencies
type AuthRuleResolver struct {
	canalRepo CanalStorage
}

func NewAuthRuleResolver(canalRepo CanalStorage) *AuthRuleResolver {
	return &AuthRuleResolver{canalRepo: canalRepo}
}

func (r *AuthRuleResolver) ResolveEventDependencies(
	ctx context.Context,
	roomID, sender string,
	eventType string,
	stateKey *string,
) ([]string, []string, error) {

	// 1. Resolve PrevEvents (The tips of the DAG)
	// This is simply querying the room_forward_extremities table.
	prevEvents, err := r.canalRepo.GetForwardExtremities(ctx, roomID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get extremities: %w", err)
	}

	// 2. Resolve AuthEvents (The VIP Pass)
	// The exact state events needed change based on what kind of event is being sent.
	// But almost ALL events require at least the create, power_levels, and sender's membership.

	authEvents := make([]string, 0)

	// Helper to fetch and append state safely
	appendState := func(stateType, stateKey string) {
		if eventID, found := r.canalRepo.GetStateEventID(ctx, roomID, stateType, stateKey); found {
			authEvents = append(authEvents, eventID)
		}
	}

	// ALWAYS REQUIRED: The room creation event
	appendState("m.room.create", "")

	// ALWAYS REQUIRED: The room power levels
	appendState("m.room.power_levels", "")

	// ALWAYS REQUIRED: The sender's membership (proving they are in the room)
	appendState("m.room.member", sender)

	// CONDITIONAL: If they are inviting someone, we also need the invitee's current membership state
	if eventType == "m.room.member" && stateKey != nil {
		// (Assuming you passed the target stateKey into this function)
		appendState("m.room.member", *stateKey)

		// And we need the join_rules to see if invites are allowed
		appendState("m.room.join_rules", "")
	}

	return prevEvents, authEvents, nil
}
