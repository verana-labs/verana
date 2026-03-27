package types

import (
	"fmt"
	"regexp"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cstypes "github.com/verana-labs/verana/x/cs/types"
)

var iso4217Regex = regexp.MustCompile(`^[A-Z]{3}$`)

// ValidateBasic performs stateless validation on MsgCreateExchangeRate.
func (msg *MsgCreateExchangeRate) ValidateBasic() error {
	// Validate authority address
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}

	// Validate base_asset_type is valid (1, 2, or 3)
	if err := validateAssetType(msg.BaseAssetType); err != nil {
		return fmt.Errorf("invalid base_asset_type: %w", err)
	}

	// Validate quote_asset_type is valid (1, 2, or 3)
	if err := validateAssetType(msg.QuoteAssetType); err != nil {
		return fmt.Errorf("invalid quote_asset_type: %w", err)
	}

	// Validate base_asset non-empty
	if msg.BaseAsset == "" {
		return fmt.Errorf("base_asset must be non-empty")
	}

	// Validate quote_asset non-empty
	if msg.QuoteAsset == "" {
		return fmt.Errorf("quote_asset must be non-empty")
	}

	// Validate asset values based on type
	if err := validateAssetValue(msg.BaseAssetType, msg.BaseAsset, "base"); err != nil {
		return err
	}
	if err := validateAssetValue(msg.QuoteAssetType, msg.QuoteAsset, "quote"); err != nil {
		return err
	}

	// The pair (base_asset_type, base_asset, quote_asset_type, quote_asset) MUST NOT be identical on both sides
	if msg.BaseAssetType == msg.QuoteAssetType && msg.BaseAsset == msg.QuoteAsset {
		return fmt.Errorf("base and quote asset pair must not be identical")
	}

	// rate MUST be a base-10 encoded unsigned integer string, strictly greater than "0"
	rate, ok := math.NewIntFromString(msg.Rate)
	if !ok {
		return fmt.Errorf("invalid rate: must be a base-10 encoded unsigned integer string")
	}
	if !rate.IsPositive() {
		return fmt.Errorf("invalid rate: must be strictly greater than 0")
	}

	// rate_scale MUST be <= 18
	if msg.RateScale > 18 {
		return fmt.Errorf("rate_scale must be <= 18")
	}

	// validity_duration MUST be >= 1 minute
	if msg.ValidityDuration < time.Minute {
		return fmt.Errorf("validity_duration must be >= 1 minute")
	}

	return nil
}

// ValidateBasic performs stateless validation on MsgUpdateExchangeRate.
func (msg *MsgUpdateExchangeRate) ValidateBasic() error {
	// Validate authority address
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}

	// Validate operator address
	if _, err := sdk.AccAddressFromBech32(msg.Operator); err != nil {
		return fmt.Errorf("invalid operator address: %w", err)
	}

	// id must be > 0
	if msg.Id == 0 {
		return fmt.Errorf("id must be greater than 0")
	}

	// rate MUST be a base-10 encoded unsigned integer string, strictly greater than "0"
	rate, ok := math.NewIntFromString(msg.Rate)
	if !ok {
		return fmt.Errorf("invalid rate: must be a base-10 encoded unsigned integer string")
	}
	if !rate.IsPositive() {
		return fmt.Errorf("invalid rate: must be strictly greater than 0")
	}

	return nil
}

// ValidateBasic performs stateless validation on MsgToggleExchangeRateState.
func (msg *MsgToggleExchangeRateState) ValidateBasic() error {
	// Validate authority address
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}

	// id must be > 0
	if msg.Id == 0 {
		return fmt.Errorf("id must be greater than 0")
	}

	return nil
}

func validateAssetType(at cstypes.PricingAssetType) error {
	switch at {
	case cstypes.PricingAssetType_TU,
		cstypes.PricingAssetType_COIN,
		cstypes.PricingAssetType_FIAT:
		return nil
	default:
		return fmt.Errorf("must be TU, COIN, or FIAT")
	}
}

func validateAssetValue(at cstypes.PricingAssetType, asset string, side string) error {
	switch at {
	case cstypes.PricingAssetType_TU:
		if asset != "TU" {
			return fmt.Errorf("%s_asset must equal \"TU\" when %s_asset_type is TRUST_UNIT", side, side)
		}
	case cstypes.PricingAssetType_COIN:
		if err := sdk.ValidateDenom(asset); err != nil {
			return fmt.Errorf("invalid %s_asset denom: %w", side, err)
		}
	case cstypes.PricingAssetType_FIAT:
		if !iso4217Regex.MatchString(asset) {
			return fmt.Errorf("%s_asset must be a valid ISO-4217 currency code (3 uppercase letters)", side)
		}
	}
	return nil
}
