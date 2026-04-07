# Verana Module Patterns

## Module Layout (all modules in x/)
```
x/<module>/
├── keeper/keeper.go <entity>.go <entity>_test.go
├── types/types.go errors.go keys.go params.go codec.go
└── module.go genesis.go msgs.go query.go [ante.go]
```

## Keeper — binary keys, never string keys
```go
func (k Keeper) TrustRegistryKey(did string) []byte {
    return append(KeyPrefixTrustRegistry, []byte(did)...)
}
func (k Keeper) SetTrustRegistry(ctx sdk.Context, tr types.TrustRegistry) {
    ctx.KVStore(k.storeKey).Set(k.TrustRegistryKey(tr.Did), k.cdc.MustMarshal(&tr))
}
func (k Keeper) GetTrustRegistry(ctx sdk.Context, did string) (types.TrustRegistry, bool) {
    bz := ctx.KVStore(k.storeKey).Get(k.TrustRegistryKey(did))
    if bz == nil { return types.TrustRegistry{}, false }
    var tr types.TrustRegistry
    k.cdc.MustUnmarshal(bz, &tr)
    return tr, true
}
```

## Errors
```go
var ErrTrustRegistryNotFound = sdkerrors.Register(ModuleName, 1, "trust registry not found")
return nil, sdkerrors.Wrapf(types.ErrTrustRegistryNotFound, "did: %s", msg.Did)
```

## Events — emit on every state change
```go
ctx.EventManager().EmitEvent(sdk.NewEvent(
    types.EventTypeTrustRegistryCreated,
    sdk.NewAttribute(types.AttributeKeyDID, tr.Did),
))
```

## Fees — sdkmath.Int only, never float64
```go
fee := sdk.NewCoin("uvna", sdkmath.NewInt(amount))
k.bankKeeper.SendCoinsFromAccountToModule(ctx, sender, types.ModuleName, sdk.NewCoins(fee))
```

## Proto — edit .proto → make proto-gen — NEVER edit generated files
## Time — ctx.BlockTime() only, NEVER time.Now()
## Genesis — export ALL KV store state
## Proto message pattern:
```protobuf
message MsgCreateTrustRegistry {
  option (cosmos.msg.v1.signer) = "creator";
  string creator = 1 [(cosmos_proto.scalar) = "cosmos.AddressString"];
  string did     = 2;
}
message MsgCreateTrustRegistryResponse { uint64 id = 1; }
```
