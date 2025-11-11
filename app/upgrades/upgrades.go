package upgrades

import (
	"github.com/verana-labs/verana/app/upgrades/types"
	v9 "github.com/verana-labs/verana/app/upgrades/v9"
)

var Upgrades = []types.Upgrade{
	v9.Upgrade,
}
