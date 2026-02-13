package v092

import (
	store "cosmossdk.io/store/types"
	"github.com/verana-labs/verana/app/upgrades/types"
)

const UpgradeName = "v0.9.2"

var Upgrade = types.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added:   []string{},
		Deleted: []string{},
	},
}
