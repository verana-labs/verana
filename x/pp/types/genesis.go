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

	// Check for duplicate perm IDs
	permissionIds := make(map[uint64]bool)
	maxPermId := uint64(0)

	for _, perm := range gs.Participants {
		// Check if ID exists
		if perm.Id == 0 {
			return fmt.Errorf("perm ID cannot be 0")
		}

		// Check for duplicate IDs
		if _, exists := permissionIds[perm.Id]; exists {
			return fmt.Errorf("duplicate perm ID found: %d", perm.Id)
		}
		permissionIds[perm.Id] = true

		// Track highest perm ID
		if perm.Id > maxPermId {
			maxPermId = perm.Id
		}

		// Validate each perm
		if err := validatePermission(perm, gs.Participants); err != nil {
			return err
		}

		// Validate timestamps are chronologically consistent
		if err := validatePermissionTimestamps(perm); err != nil {
			return err
		}
	}

	// Check for duplicate session IDs
	sessionIds := make(map[string]bool)
	for _, session := range gs.ParticipantSessions {
		// Check if ID exists
		if session.Id == "" {
			return fmt.Errorf("perm session ID cannot be empty")
		}

		// Check for duplicate IDs
		if _, exists := sessionIds[session.Id]; exists {
			return fmt.Errorf("duplicate perm session ID found: %s", session.Id)
		}
		sessionIds[session.Id] = true

		// Validate perm references
		if err := validateParticipantSession(session, permissionIds); err != nil {
			return err
		}
	}

	// Validate next perm ID is greater than max perm ID
	if len(gs.Participants) > 0 && gs.NextParticipantId <= maxPermId {
		return fmt.Errorf("next_permission_id (%d) must be greater than the maximum perm ID (%d)",
			gs.NextParticipantId, maxPermId)
	}

	return nil
}

// validatePermission validates a single perm
func validatePermission(perm Participant, allPerms []Participant) error {
	// Check required fields
	if perm.Role == 0 {
		return fmt.Errorf("role cannot be 0 for participant ID %d", perm.Id)
	}

	if perm.CorporationId == 0 {
		return fmt.Errorf("corporation_id cannot be 0 for participant ID %d", perm.Id)
	}

	// did is mandatory per spec v4-rc2
	if perm.Did == "" {
		return fmt.Errorf("did is mandatory for participant ID %d", perm.Id)
	}

	// op_state is mandatory per spec v4-rc2 (PENDING/VALIDATED/TERMINATED)
	if perm.OpState == OnboardingState_ONBOARDING_STATE_UNSPECIFIED {
		return fmt.Errorf("op_state cannot be unspecified for participant ID %d", perm.Id)
	}

	// Validate validator perm reference
	if perm.ValidatorParticipantId != 0 {
		validatorFound := false

		// Check if validator perm exists
		for _, p := range allPerms {
			if p.Id == perm.ValidatorParticipantId {
				validatorFound = true
				break
			}
		}

		if !validatorFound {
			return fmt.Errorf("validator perm ID %d not found for perm ID %d",
				perm.ValidatorParticipantId, perm.Id)
		}
	}

	return nil
}

// validatePermissionTimestamps validates that timestamps are chronologically consistent
func validatePermissionTimestamps(perm Participant) error {
	// Check that modified time exists
	if perm.Modified == nil {
		return fmt.Errorf("modified timestamp is required for perm ID %d", perm.Id)
	}

	// Check that created time exists
	if perm.Created == nil {
		return fmt.Errorf("created timestamp is required for perm ID %d", perm.Id)
	}

	// op_last_state_change is mandatory per spec v4-rc2
	if perm.OpLastStateChange == nil {
		return fmt.Errorf("op_last_state_change is required for participant ID %d", perm.Id)
	}

	// If effective_from and effective_until both exist, ensure effective_from is before effective_until
	if perm.EffectiveFrom != nil && perm.EffectiveUntil != nil {
		if !perm.EffectiveFrom.Before(*perm.EffectiveUntil) {
			return fmt.Errorf("effective_from must be before effective_until for perm ID %d", perm.Id)
		}
	}

	// If adjusted time exists, it should be after created time
	if perm.Adjusted != nil && perm.Created != nil {
		if !perm.Created.Before(*perm.Adjusted) {
			return fmt.Errorf("adjusted timestamp must be after created timestamp for perm ID %d", perm.Id)
		}
	}

	return nil
}

// validateParticipantSession validates a single perm session
func validateParticipantSession(session ParticipantSession, permissionIds map[uint64]bool) error {
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
			return fmt.Errorf("at least one of issuer_perm_id or verifier_perm_id must be set for session ID %s, record index %d",
				session.Id, i)
		}

		// Check that issuer perm exists if set
		if record.IssuerParticipantId != 0 && !permissionIds[record.IssuerParticipantId] {
			return fmt.Errorf("issuer perm ID %d not found for session ID %s, record index %d",
				record.IssuerParticipantId, session.Id, i)
		}

		// Check that verifier perm exists if set
		if record.VerifierParticipantId != 0 && !permissionIds[record.VerifierParticipantId] {
			return fmt.Errorf("verifier perm ID %d not found for session ID %s, record index %d",
				record.VerifierParticipantId, session.Id, i)
		}

		// Check that wallet agent perm exists if set
		if record.WalletAgentParticipantId != 0 && !permissionIds[record.WalletAgentParticipantId] {
			return fmt.Errorf("wallet agent participant ID %d not found for session ID %s, record index %d",
				record.WalletAgentParticipantId, session.Id, i)
		}

		// Check that agent participant exists if set
		if record.AgentParticipantId != 0 && !permissionIds[record.AgentParticipantId] {
			return fmt.Errorf("agent participant ID %d not found for session ID %s, record index %d",
				record.AgentParticipantId, session.Id, i)
		}
	}

	return nil
}
