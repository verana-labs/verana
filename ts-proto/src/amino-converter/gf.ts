import type { AminoConverter } from "@cosmjs/stargate";
import {
  MsgAddGovernanceFrameworkDocument,
  MsgIncreaseActiveGovernanceFrameworkVersion,
} from "../codec/verana/gf/v1/tx";
import { clean, u64ToStr } from "./util/helpers";

// MOD-GF Msgs extracted from x/tr in v4-rc2. ecosystem_id is the optional
// polymorphic subject discriminator: when zero/absent, the target is the
// signing corporation's own CGF; when set, the target is that Ecosystem.

export const MsgAddGovernanceFrameworkDocumentAminoConverter: AminoConverter = {
  aminoType: "verana/x/gf/MsgAddGovernanceFrameworkDocument",
  toAmino: ({
    corporation,
    operator,
    ecosystemId,
    docLanguage,
    docUrl,
    docDigestSri,
    version,
  }: MsgAddGovernanceFrameworkDocument) => clean({
    corporation: corporation || undefined,
    operator: operator || undefined,
    ecosystem_id: u64ToStr(ecosystemId as any),
    doc_language: docLanguage || undefined,
    doc_url: docUrl || undefined,
    doc_digest_sri: docDigestSri || undefined,
    version: version || undefined,
  }),
  fromAmino: (value: any) =>
    MsgAddGovernanceFrameworkDocument.fromPartial({
      corporation: value.corporation ?? "",
      operator: value.operator ?? "",
      ecosystemId: value.ecosystem_id != null ? Number(value.ecosystem_id) : 0,
      docLanguage: value.doc_language ?? "",
      docUrl: value.doc_url ?? "",
      docDigestSri: value.doc_digest_sri ?? "",
      version: value.version ?? 0,
    }),
};

export const MsgIncreaseActiveGovernanceFrameworkVersionAminoConverter: AminoConverter = {
  aminoType: "verana/x/gf/MsgIncreaseActiveGovernanceFrameworkVersion",
  toAmino: ({ corporation, operator, ecosystemId }: MsgIncreaseActiveGovernanceFrameworkVersion) => clean({
    corporation: corporation || undefined,
    operator: operator || undefined,
    ecosystem_id: u64ToStr(ecosystemId as any),
  }),
  fromAmino: (value: any) =>
    MsgIncreaseActiveGovernanceFrameworkVersion.fromPartial({
      corporation: value.corporation ?? "",
      operator: value.operator ?? "",
      ecosystemId: value.ecosystem_id != null ? Number(value.ecosystem_id) : 0,
    }),
};
