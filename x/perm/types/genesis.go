package types

import (
	"fmt"
)

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:             DefaultParams(),
		Permissions:        []Permission{},
		PermissionSessions: []PermissionSession{},
		NextPermissionId:   0,
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

	for _, perm := range gs.Permissions {
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
		if err := validatePermission(perm, gs.Permissions); err != nil {
			return err
		}

		// Validate timestamps are chronologically consistent
		if err := validatePermissionTimestamps(perm); err != nil {
			return err
		}
	}

	// Check for duplicate session IDs
	sessionIds := make(map[string]bool)
	for _, session := range gs.PermissionSessions {
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
		if err := validatePermissionSession(session, permissionIds); err != nil {
			return err
		}
	}

	// Validate next perm ID is greater than max perm ID
	if len(gs.Permissions) > 0 && gs.NextPermissionId <= maxPermId {
		return fmt.Errorf("next_permission_id (%d) must be greater than the maximum perm ID (%d)",
			gs.NextPermissionId, maxPermId)
	}

	return nil
}

// validatePermission validates a single perm
func validatePermission(perm Permission, allPerms []Permission) error {
	// Check required fields
	if perm.Type == 0 {
		return fmt.Errorf("perm type cannot be 0 for perm ID %d", perm.Id)
	}

	if perm.Authority == "" {
		return fmt.Errorf("authority cannot be empty for perm ID %d", perm.Id)
	}

	// Validate validator perm reference
	if perm.ValidatorPermId != 0 {
		validatorFound := false

		// Check if validator perm exists
		for _, p := range allPerms {
			if p.Id == perm.ValidatorPermId {
				validatorFound = true
				break
			}
		}

		if !validatorFound {
			return fmt.Errorf("validator perm ID %d not found for perm ID %d",
				perm.ValidatorPermId, perm.Id)
		}
	}

	return nil
}

// validatePermissionTimestamps validates that timestamps are chronologically consistent
func validatePermissionTimestamps(perm Permission) error {
	// Check that modified time exists
	if perm.Modified == nil {
		return fmt.Errorf("modified timestamp is required for perm ID %d", perm.Id)
	}

	// Check that created time exists
	if perm.Created == nil {
		return fmt.Errorf("created timestamp is required for perm ID %d", perm.Id)
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

// validatePermissionSession validates a single perm session
func validatePermissionSession(session PermissionSession, permissionIds map[uint64]bool) error {
	// Check that agent perm exists
	if session.AgentPermId == 0 {
		return fmt.Errorf("agent perm ID cannot be 0 for session ID %s", session.Id)
	}

	if !permissionIds[session.AgentPermId] {
		return fmt.Errorf("agent perm ID %d not found for session ID %s", session.AgentPermId, session.Id)
	}

	// Validate timestamps
	if session.Created == nil {
		return fmt.Errorf("created timestamp is required for session ID %s", session.Id)
	}

	if session.Modified == nil {
		return fmt.Errorf("modified timestamp is required for session ID %s", session.Id)
	}

	// Validate each session record
	for i, record := range session.SessionRecords {
		// At least one of issuer or verifier must be set
		if record.IssuerPermId == 0 && record.VerifierPermId == 0 {
			return fmt.Errorf("at least one of issuer_perm_id or verifier_perm_id must be set for session ID %s, record index %d",
				session.Id, i)
		}

		// Check that issuer perm exists if set
		if record.IssuerPermId != 0 && !permissionIds[record.IssuerPermId] {
			return fmt.Errorf("issuer perm ID %d not found for session ID %s, record index %d",
				record.IssuerPermId, session.Id, i)
		}

		// Check that verifier perm exists if set
		if record.VerifierPermId != 0 && !permissionIds[record.VerifierPermId] {
			return fmt.Errorf("verifier perm ID %d not found for session ID %s, record index %d",
				record.VerifierPermId, session.Id, i)
		}

		// Check that wallet agent perm exists if set
		if record.WalletAgentPermId != 0 && !permissionIds[record.WalletAgentPermId] {
			return fmt.Errorf("wallet agent perm ID %d not found for session ID %s, record index %d",
				record.WalletAgentPermId, session.Id, i)
		}
	}

	return nil
}
