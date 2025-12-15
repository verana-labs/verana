# @verana/proto-codecs

TypeScript protobuf codecs generated from the Verana blockchain protobufs with `ts-proto`.

## Generate locally
From the repository root:

```bash
cd proto
buf generate --template buf.gen.ts.yaml    # writes to ../ts-proto/src/codec
cd ../ts-proto
npm install
npm run build                              # emits dist/ for publishing
```

You can also use `make proto-ts` once dependencies are installed.

## Publish
1. Bump the version in `package.json` to match the chain tag you are releasing (e.g. `v0.7.0` -> `0.7.0`).
2. Regenerate and build as above.
3. `npm publish` (or `npm pack` and upload the tarball to the GitHub release assets).

## Consume

```bash
npm install @verana/proto-codecs
```

Import directly from the codec paths:

```ts
import { MsgAddDID } from "@verana/proto-codecs/codec/verana/dd/v1/tx";
```
