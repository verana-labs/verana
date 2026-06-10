package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:                   DefaultParams(),
		OperatorAuthorizations:   []OperatorAuthorization{},
		FeeGrants:                []FeeGrant{},
		VsOperatorAuthorizations: []VSOperatorAuthorization{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}

	// OperatorAuthorization: id (key) non-zero + unique; (corporation_id,
	// operator) secondary index unique per spec v4-rc2.
	oaIDs := make(map[uint64]bool)
	oaCorpOp := make(map[string]bool)
	for i, oa := range gs.OperatorAuthorizations {
		if oa.Id == 0 {
			return fmt.Errorf("operator_authorizations[%d]: id cannot be 0", i)
		}
		if oaIDs[oa.Id] {
			return fmt.Errorf("operator_authorizations[%d]: duplicate id %d", i, oa.Id)
		}
		oaIDs[oa.Id] = true
		if oa.CorporationId == 0 {
			return fmt.Errorf("operator_authorizations[%d]: corporation_id cannot be 0", i)
		}
		if _, err := sdk.AccAddressFromBech32(oa.Operator); err != nil {
			return fmt.Errorf("operator_authorizations[%d]: invalid operator address: %w", i, err)
		}
		if len(oa.MsgTypes) == 0 {
			return fmt.Errorf("operator_authorizations[%d]: msg_types cannot be empty", i)
		}
		idx := fmt.Sprintf("%d/%s", oa.CorporationId, oa.Operator)
		if oaCorpOp[idx] {
			return fmt.Errorf("operator_authorizations[%d]: duplicate (corporation_id, operator) %s", i, idx)
		}
		oaCorpOp[idx] = true
	}

	// FeeGrant: composite key (grantor_corporation_id, grantee) unique.
	fgKeys := make(map[string]bool)
	for i, fg := range gs.FeeGrants {
		if fg.GrantorCorporationId == 0 {
			return fmt.Errorf("fee_grants[%d]: grantor_corporation_id cannot be 0", i)
		}
		if _, err := sdk.AccAddressFromBech32(fg.Grantee); err != nil {
			return fmt.Errorf("fee_grants[%d]: invalid grantee address: %w", i, err)
		}
		if len(fg.MsgTypes) == 0 {
			return fmt.Errorf("fee_grants[%d]: msg_types cannot be empty", i)
		}
		key := fmt.Sprintf("%d/%s", fg.GrantorCorporationId, fg.Grantee)
		if fgKeys[key] {
			return fmt.Errorf("fee_grants[%d]: duplicate (grantor_corporation_id, grantee) %s", i, key)
		}
		fgKeys[key] = true
	}

	// VSOperatorAuthorization: id (key) non-zero + unique; (corporation_id,
	// vs_operator) unique; the entry exists iff it has at least one record;
	// each record's participant_id is globally unique across all VSOAs.
	vsoaIDs := make(map[uint64]bool)
	vsoaCorpOp := make(map[string]bool)
	participantIDs := make(map[uint64]bool)
	for i, vsoa := range gs.VsOperatorAuthorizations {
		if vsoa.Id == 0 {
			return fmt.Errorf("vs_operator_authorizations[%d]: id cannot be 0", i)
		}
		if vsoaIDs[vsoa.Id] {
			return fmt.Errorf("vs_operator_authorizations[%d]: duplicate id %d", i, vsoa.Id)
		}
		vsoaIDs[vsoa.Id] = true
		if vsoa.CorporationId == 0 {
			return fmt.Errorf("vs_operator_authorizations[%d]: corporation_id cannot be 0", i)
		}
		if _, err := sdk.AccAddressFromBech32(vsoa.VsOperator); err != nil {
			return fmt.Errorf("vs_operator_authorizations[%d]: invalid vs_operator address: %w", i, err)
		}
		idx := fmt.Sprintf("%d/%s", vsoa.CorporationId, vsoa.VsOperator)
		if vsoaCorpOp[idx] {
			return fmt.Errorf("vs_operator_authorizations[%d]: duplicate (corporation_id, vs_operator) %s", i, idx)
		}
		vsoaCorpOp[idx] = true
		if len(vsoa.Records) == 0 {
			return fmt.Errorf("vs_operator_authorizations[%d]: must have at least one record", i)
		}
		for j, rec := range vsoa.Records {
			if rec.ParticipantId == 0 {
				return fmt.Errorf("vs_operator_authorizations[%d].records[%d]: participant_id cannot be 0", i, j)
			}
			if participantIDs[rec.ParticipantId] {
				return fmt.Errorf("vs_operator_authorizations[%d].records[%d]: duplicate participant_id %d (must be globally unique)", i, j, rec.ParticipantId)
			}
			participantIDs[rec.ParticipantId] = true
			if len(rec.MsgTypes) == 0 {
				return fmt.Errorf("vs_operator_authorizations[%d].records[%d]: msg_types cannot be empty", i, j)
			}
		}
	}

	return nil
}
