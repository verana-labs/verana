package v6

import (
	store "cosmossdk.io/store/types"
	"github.com/verana-labs/verana/app/upgrades/types"
)

const UpgradeName = "v0.6"

var Upgrade = types.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{
			"cs",
			"dd",
			"perm",
			"td",
			"tr",
		},
		Deleted: []string{},
	},
}
