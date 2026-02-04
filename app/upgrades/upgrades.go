package upgrades

import (
	"github.com/verana-labs/verana/app/upgrades/types"
	v091 "github.com/verana-labs/verana/app/upgrades/v091"
	v092 "github.com/verana-labs/verana/app/upgrades/v092"
	v093 "github.com/verana-labs/verana/app/upgrades/v093"
	v9 "github.com/verana-labs/verana/app/upgrades/v9"
)

var Upgrades = []types.Upgrade{
	v093.Upgrade,
	v092.Upgrade,
	v9.Upgrade,
	v091.Upgrade,
}
