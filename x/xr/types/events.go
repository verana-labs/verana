package types

const (
	EventTypeCreateExchangeRate         = "create_exchange_rate"
	EventTypeUpdateExchangeRate         = "update_exchange_rate"
	EventTypeToggleExchangeRateState    = "toggle_exchange_rate_state"

	AttributeKeyID             = "id"
	AttributeKeyBaseAssetType  = "base_asset_type"
	AttributeKeyBaseAsset      = "base_asset"
	AttributeKeyQuoteAssetType = "quote_asset_type"
	AttributeKeyQuoteAsset     = "quote_asset"
	AttributeKeyAuthority      = "authority"
	AttributeKeyOperator       = "operator"
	AttributeKeyRate           = "rate"
	AttributeKeyState          = "state"
)
