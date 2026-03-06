package lib

import (
	"bytes"
	"encoding/json"
	"log"

	"github.com/cosmos/gogoproto/proto"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"
	"github.com/verana-labs/verana/x/tr/types"
)

// PrettyJSON formats a proto message as indented JSON
func PrettyJSON(client cosmosclient.Client, protoObj proto.Message) string {
	jsonResp, err := client.Context().Codec.MarshalJSON(protoObj)
	if err != nil {
		log.Fatal(err)
	}
	var prettyJSON bytes.Buffer
	err = json.Indent(&prettyJSON, jsonResp, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	return prettyJSON.String()
}

// IsDidUsed checks if a DID is already used in the registry
func IsDidUsed(listRegs *types.QueryListTrustRegistriesResponse, did string) bool {
	for _, reg := range listRegs.TrustRegistries {
		if reg.Did == did {
			return true
		}
	}
	return false
}
