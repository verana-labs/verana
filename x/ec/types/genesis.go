package types

import (
	"cosmossdk.io/errors"

	"github.com/verana-labs/verana/util/validation"
)

// DefaultGenesis returns the default genesis state.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:     DefaultParams(),
		Ecosystems: []Ecosystem{},
		Counters:   []Counter{},
	}
}

// Validate enforces at-rest invariants for the EC module:
//   - per-entry shape (id, corporation_id, language, timestamps);
//   - duplicate-id rejection;
//   - per-Ecosystem (did, corporation_id) consistency invariant
//     (MOD-ES entity spec);
//   - counter sanity (>= max(ecosystem.id)).
//
// GFV/GFD live in x/gf and MUST NOT be exported from this module's genesis.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}

	seenIDs := make(map[uint64]struct{}, len(gs.Ecosystems))
	// didOwner maps did → corporation_id; second hit with different owner
	// violates the per-Ecosystem (did, corp_id) consistency invariant.
	didOwner := make(map[string]uint64)

	for i, ec := range gs.Ecosystems {
		if ec.Id == 0 {
			return errors.Wrapf(ErrInvalidSubject, "ecosystem at index %d: id must be > 0", i)
		}
		if _, dup := seenIDs[ec.Id]; dup {
			return errors.Wrapf(ErrInvalidSubject, "duplicate ecosystem id %d", ec.Id)
		}
		seenIDs[ec.Id] = struct{}{}

		if ec.CorporationId == 0 {
			return errors.Wrapf(ErrInvalidSubject, "ecosystem %d: corporation_id must be > 0", ec.Id)
		}
		if ec.Did == "" {
			return errors.Wrapf(ErrInvalidDID, "ecosystem %d: did is required", ec.Id)
		}
		if !validation.IsValidDID(ec.Did) {
			return errors.Wrapf(ErrInvalidDID, "ecosystem %d: did syntax invalid", ec.Id)
		}
		if ec.Language == "" {
			return errors.Wrapf(ErrInvalidLanguage, "ecosystem %d: language is required", ec.Id)
		}
		if !IsValidBCP47(ec.Language) {
			return errors.Wrapf(ErrInvalidLanguage, "ecosystem %d: language must be BCP 47", ec.Id)
		}
		if ec.Created.IsZero() {
			return errors.Wrapf(ErrInvalidTimestamp, "ecosystem %d: created is required", ec.Id)
		}
		if ec.Modified.IsZero() {
			return errors.Wrapf(ErrInvalidTimestamp, "ecosystem %d: modified is required", ec.Id)
		}
		if ec.Modified.Before(ec.Created) {
			return errors.Wrapf(ErrInvalidTimestamp, "ecosystem %d: modified (%s) is before created (%s)", ec.Id, ec.Modified, ec.Created)
		}
		if ec.ActiveVersion == 0 {
			return errors.Wrapf(ErrInvalidSubject, "ecosystem %d: active_version must be >= 1", ec.Id)
		}

		if owner, seen := didOwner[ec.Did]; seen {
			if owner != ec.CorporationId {
				return errors.Wrapf(ErrDIDOwnershipConflict, "did %q owned by corporation %d but ecosystem %d claims corporation %d", ec.Did, owner, ec.Id, ec.CorporationId)
			}
		} else {
			didOwner[ec.Did] = ec.CorporationId
		}
	}

	// Counter sanity: only "ec" is meaningful here.
	seenCounter := make(map[string]struct{}, len(gs.Counters))
	var maxEcID uint64
	for _, ec := range gs.Ecosystems {
		if ec.Id > maxEcID {
			maxEcID = ec.Id
		}
	}
	for _, c := range gs.Counters {
		if _, dup := seenCounter[c.EntityType]; dup {
			return errors.Wrapf(ErrInvalidSubject, "duplicate counter entity_type %q", c.EntityType)
		}
		seenCounter[c.EntityType] = struct{}{}
		if c.EntityType != "ec" {
			return errors.Wrapf(ErrInvalidSubject, "unknown counter entity_type %q (only \"ec\" is allowed)", c.EntityType)
		}
		if c.Value < maxEcID {
			return errors.Wrapf(ErrInvalidSubject, "ec counter (%d) is less than max ecosystem id (%d)", c.Value, maxEcID)
		}
	}
	return nil
}
