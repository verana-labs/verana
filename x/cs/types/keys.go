package types

import (
	"cosmossdk.io/collections"
)

const (
	// ModuleName defines the module name
	ModuleName = "cs"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_credentialschema"

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName
)

var (
	ParamsKey                    = []byte("p_credentialschema")
	CredentialSchemaKey          = collections.NewPrefix(1)
	CounterKey                   = collections.NewPrefix(2)
	SchemaAuthorizationPolicyKey = collections.NewPrefix(3)
)

const CounterKeySchemaAuthorizationPolicy = "schema_authorization_policy"

func KeyPrefix(p string) []byte {
	return []byte(p)
}
