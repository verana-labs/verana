# Amino Converters

Amino converters translate between Protobuf messages and Amino JSON format, which is required for Amino (LEGACY_AMINO_JSON) signing with CosmJS.

## Structure

```
amino-converter/
  cs.ts          # Credential Schema module converters
  dd.ts          # DID Directory module converters
  perm.ts        # Permission module converters
  td.ts          # Trust Deposit module converters
  tr.ts          # Trust Registry module converters
  util/
    helpers.ts   # Shared conversion helpers
```

## Helper Reference

When creating a new Amino converter, use the helpers from `./util/helpers` based on the field types in the Protobuf message:

| Proto Field Type | toAmino (Proto -> JSON) | fromAmino (JSON -> Proto) | Notes |
|---|---|---|---|
| `uint64` / `Long` | `u64ToStr(value)` | `strToU64(value)` | Amino encodes uint64 as string |
| `uint32` | `u32ToAmino(value)` | direct assignment | Preserves 0 as `0`, omits `null` |
| `OptionalUInt32` | `toOptU32Amino(value)` | `fromOptU32Amino(value)` | 0 -> `{}`, n -> `{value: n}`, absent -> `undefined` |
| `google.protobuf.Timestamp` / `Date` | `dateToIsoAmino(value)` | `isoToDate(value)` | ISO 8601 string, trims `.000Z` to `Z` |
| `string` | direct assignment | direct assignment | No conversion needed |
| `bool` | `value ? true : undefined` | `value ?? false` | Omit `false` for omitempty |
| `string` (optional) | `value ?? ''` | `value ?? ''` | Use empty string default |

### Additional helpers

- **`clean(obj)`**: Removes `undefined` fields from an object. Use when fields should be omitted (omitempty) rather than sent as `null`.
- **`pickOptionalUInt32(value)`**: Parses loosely-typed input (string, number) into an `OptionalUInt32` wrapper.

## Creating a New Amino Converter

1. Create a new file in `amino-converter/` named after the module (e.g., `mymodule.ts`).
2. Import the Protobuf message types from the codec: `../codec/verana/<module>/v1/tx`.
3. Import helpers from `./util/helpers`.
4. For each message type, export a converter object with:
   - `aminoType`: the full proto type URL (e.g., `'/verana.mymodule.v1.MsgDoSomething'`)
   - `toAmino(msg)`: converts Proto message to Amino JSON object
   - `fromAmino(value)`: converts Amino JSON back to Proto message using `Msg.fromPartial()`

### Field naming convention

- Proto uses **camelCase** (e.g., `schemaId`, `docUrl`)
- Amino JSON uses **snake_case** (e.g., `schema_id`, `doc_url`)

### Example

```typescript
import { MsgDoSomething } from '../codec/verana/mymodule/v1/tx';
import { u64ToStr, strToU64, clean } from './util/helpers';

export const MsgDoSomethingAminoConverter = {
  aminoType: '/verana.mymodule.v1.MsgDoSomething',
  toAmino: (msg: MsgDoSomething) => clean({
    creator: msg.creator ?? '',
    some_id: u64ToStr(msg.someId),       // uint64 -> string
  }),
  fromAmino: (value: any) =>
    MsgDoSomething.fromPartial({
      creator: value.creator ?? '',
      someId: strToU64(value.some_id),   // string -> uint64
    }),
};
```

## Proto Codecs

Generated with:
- `protoc-gen-ts_proto` v1.181.2
- `protoc` v5.29.3 (libprotoc 29.3)
