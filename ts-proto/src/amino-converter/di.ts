import type { AminoConverter } from "@cosmjs/stargate";
import { MsgStoreDigest } from "../codec/verana/di/v1/tx";
import { clean } from "./util/helpers";

export const MsgStoreDigestAminoConverter: AminoConverter = {
  aminoType: "verana/x/di/MsgStoreDigest",
  toAmino: (m: MsgStoreDigest) => clean({
    authority: m.authority || undefined,
    operator: m.operator || undefined,
    digest: m.digest || undefined,
    digest_algorithm: m.digestAlgorithm || undefined,
  }),
  fromAmino: (a: any): MsgStoreDigest =>
    MsgStoreDigest.fromPartial({
      authority: a.authority ?? "",
      operator: a.operator ?? "",
      digest: a.digest ?? "",
      digestAlgorithm: a.digest_algorithm ?? "",
    }),
};
