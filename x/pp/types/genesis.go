package types

import (
	"fmt"
)

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:              DefaultParams(),
		Participants:        []Participant{},
		ParticipantSessions: []ParticipantSession{},
		NextParticipantId:   0,
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// Validate params
	if err := gs.Params.Validate(); err != nil {
		return err
	}

	// Check for duplicate participant IDs
	participantIds := make(map[uint64]bool)
	maxParticipantId := uint64(0)

	for _, participant := range gs.Participants {
		// Check if ID exists
		if participant.Id == 0 {
			return fmt.Errorf("participant ID cannot be 0")
		}

		// Check for duplicate IDs
		if _, exists := participantIds[participant.Id]; exists {
			return fmt.Errorf("duplicate participant ID found: %d", participant.Id)
		}
		participantIds[participant.Id] = true

		// Track highest participant ID
		if participant.Id > maxParticipantId {
			maxParticipantId = participant.Id
		}

		// Validate each participant
		if err := validateParticipant(participant, gs.Participants); err != nil {
			return err
		}

		// Validate timestamps are chronologically consistent
		if err := validateParticipantTimestamps(participant); err != nil {
			return err
		}
	}

	// Check for duplicate session IDs
	sessionIds := make(map[string]bool)
	for _, session := range gs.ParticipantSessions {
		// Check if ID exists
		if session.Id == "" {
			return fmt.Errorf("participant session ID cannot be empty")
		}

		// Check for duplicate IDs
		if _, exists := sessionIds[session.Id]; exists {
			return fmt.Errorf("duplicate participant session ID found: %s", session.Id)
		}
		sessionIds[session.Id] = true

		// Validate participant references
		if err := validateParticipantSession(session, participantIds); err != nil {
			return err
		}
	}

	// Validate next participant ID is greater than max participant ID
	if len(gs.Participants) > 0 && gs.NextParticipantId <= maxParticipantId {
		return fmt.Errorf("next_participant_id (%d) must be greater than the maximum participant ID (%d)",
			gs.NextParticipantId, maxParticipantId)
	}

	return nil
}

// validateParticipant validates a single participant
func validateParticipant(participant Participant, allParticipants []Participant) error {
	// Check required fields
	if participant.Role == 0 {
		return fmt.Errorf("role cannot be 0 for participant ID %d", participant.Id)
	}

	if participant.CorporationId == 0 {
		return fmt.Errorf("corporation_id cannot be 0 for participant ID %d", participant.Id)
	}

	// did is mandatory per spec v4-rc2
	if participant.Did == "" {
		return fmt.Errorf("did is mandatory for participant ID %d", participant.Id)
	}

	// op_state is mandatory per spec v4-rc2 (PENDING/VALIDATED/TERMINATED)
	if participant.OpState == OnboardingState_ONBOARDING_STATE_UNSPECIFIED {
		return fmt.Errorf("op_state cannot be unspecified for participant ID %d", participant.Id)
	}

	// Validate validator participant reference
	if participant.ValidatorParticipantId != 0 {
		validatorFound := false

		// Check if validator participant exists
		for _, p := range allParticipants {
			if p.Id == participant.ValidatorParticipantId {
				validatorFound = true
				break
			}
		}

		if !validatorFound {
			return fmt.Errorf("validator participant ID %d not found for participant ID %d",
				participant.ValidatorParticipantId, participant.Id)
		}
	}

	return nil
}

// validateParticipantTimestamps validates that timestamps are chronologically consistent
func validateParticipantTimestamps(participant Participant) error {
	// Check that modified time exists
	if participant.Modified == nil {
		return fmt.Errorf("modified timestamp is required for participant ID %d", participant.Id)
	}

	// Check that created time exists
	if participant.Created == nil {
		return fmt.Errorf("created timestamp is required for participant ID %d", participant.Id)
	}

	// op_last_state_change is mandatory per spec v4-rc2
	if participant.OpLastStateChange == nil {
		return fmt.Errorf("op_last_state_change is required for participant ID %d", participant.Id)
	}

	// If effective_from and effective_until both exist, ensure effective_from is before effective_until
	if participant.EffectiveFrom != nil && participant.EffectiveUntil != nil {
		if !participant.EffectiveFrom.Before(*participant.EffectiveUntil) {
			return fmt.Errorf("effective_from must be before effective_until for participant ID %d", participant.Id)
		}
	}

	// If adjusted time exists, it should be after created time
	if participant.Adjusted != nil && participant.Created != nil {
		if !participant.Created.Before(*participant.Adjusted) {
			return fmt.Errorf("adjusted timestamp must be after created timestamp for participant ID %d", participant.Id)
		}
	}

	return nil
}

// validateParticipantSession validates a single participant session
func validateParticipantSession(session ParticipantSession, participantIds map[uint64]bool) error {
	// Validate timestamps
	if session.Created == nil {
		return fmt.Errorf("created timestamp is required for session ID %s", session.Id)
	}

	if session.Modified == nil {
		return fmt.Errorf("modified timestamp is required for session ID %s", session.Id)
	}

	// Validate each session record
	for i, record := range session.SessionRecords {
		// record id is the mandatory key per spec v4-rc2
		if record.Id == 0 {
			return fmt.Errorf("session record id cannot be 0 for session ID %s, record index %d", session.Id, i)
		}

		// At least one of issuer or verifier must be set
		if record.IssuerParticipantId == 0 && record.VerifierParticipantId == 0 {
			return fmt.Errorf("at least one of issuer_participant_id or verifier_participant_id must be set for session ID %s, record index %d",
				session.Id, i)
		}

		// Check that issuer participant exists if set
		if record.IssuerParticipantId != 0 && !participantIds[record.IssuerParticipantId] {
			return fmt.Errorf("issuer participant ID %d not found for session ID %s, record index %d",
				record.IssuerParticipantId, session.Id, i)
		}

		// Check that verifier participant exists if set
		if record.VerifierParticipantId != 0 && !participantIds[record.VerifierParticipantId] {
			return fmt.Errorf("verifier participant ID %d not found for session ID %s, record index %d",
				record.VerifierParticipantId, session.Id, i)
		}

		// Check that wallet agent participant exists if set
		if record.WalletAgentParticipantId != 0 && !participantIds[record.WalletAgentParticipantId] {
			return fmt.Errorf("wallet agent participant ID %d not found for session ID %s, record index %d",
				record.WalletAgentParticipantId, session.Id, i)
		}

		// Check that agent participant exists if set
		if record.AgentParticipantId != 0 && !participantIds[record.AgentParticipantId] {
			return fmt.Errorf("agent participant ID %d not found for session ID %s, record index %d",
				record.AgentParticipantId, session.Id, i)
		}
	}

	return nil
}
