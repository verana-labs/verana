package types

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:    DefaultParams(),
		Versions:  nil,
		Documents: nil,
	}
}

// Validate performs basic genesis state validation.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}
	// GFV: exactly one of ecosystem_id (>0) and corporation (non-empty) must be set.
	for _, gfv := range gs.Versions {
		hasEco := gfv.EcosystemId > 0
		hasCorp := gfv.CorporationId > 0
		if hasEco == hasCorp {
			return ErrInvalidSubject
		}
		if gfv.Version < 1 {
			return ErrInvalidVersion
		}
	}
	// GFD: gfv_id must reference a GFV in this genesis.
	versionIDs := map[uint64]struct{}{}
	for _, gfv := range gs.Versions {
		versionIDs[gfv.Id] = struct{}{}
	}
	for _, gfd := range gs.Documents {
		if _, ok := versionIDs[gfd.GfvId]; !ok {
			return ErrInvalidVersion
		}
		if !IsValidBCP47(gfd.Language) {
			return ErrInvalidLanguage
		}
	}
	return nil
}
