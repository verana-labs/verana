import type { AminoConverter } from "@cosmjs/stargate";
import {
  MsgCreateExchangeRate,
  MsgToggleExchangeRateState,
  MsgUpdateExchangeRate,
} from "../codec/verana/xr/v1/tx";
import {
  aminoToDuration,
  clean,
  durationToAmino,
  strToU64,
  u32ToAmino,
  u64ToStr,
} from "./util/helpers";

export const MsgCreateExchangeRateAminoConverter: AminoConverter = {
  aminoType: "/verana.xr.v1.MsgCreateExchangeRate",
  toAmino: (m: MsgCreateExchangeRate) => clean({
    authority: m.authority || undefined,
    base_asset_type: m.baseAssetType ?? 0,
    base_asset: m.baseAsset || undefined,
    quote_asset_type: m.quoteAssetType ?? 0,
    quote_asset: m.quoteAsset || undefined,
    rate: m.rate || undefined,
    rate_scale: u32ToAmino(m.rateScale),
    validity_duration: durationToAmino(m.validityDuration),
  }),
  fromAmino: (a: any): MsgCreateExchangeRate =>
    MsgCreateExchangeRate.fromPartial({
      authority: a.authority ?? "",
      baseAssetType: a.base_asset_type ?? 0,
      baseAsset: a.base_asset ?? "",
      quoteAssetType: a.quote_asset_type ?? 0,
      quoteAsset: a.quote_asset ?? "",
      rate: a.rate ?? "",
      rateScale: a.rate_scale ?? 0,
      validityDuration: aminoToDuration(a.validity_duration),
    }),
};

export const MsgUpdateExchangeRateAminoConverter: AminoConverter = {
  aminoType: "/verana.xr.v1.MsgUpdateExchangeRate",
  toAmino: (m: MsgUpdateExchangeRate) => clean({
    authority: m.authority || undefined,
    operator: m.operator || undefined,
    id: u64ToStr(m.id),
    rate: m.rate || undefined,
  }),
  fromAmino: (a: any): MsgUpdateExchangeRate =>
    MsgUpdateExchangeRate.fromPartial({
      authority: a.authority ?? "",
      operator: a.operator ?? "",
      id: strToU64(a.id) != null ? Number(strToU64(a.id)!.toString()) : 0,
      rate: a.rate ?? "",
    }),
};

export const MsgToggleExchangeRateStateAminoConverter: AminoConverter = {
  aminoType: "/verana.xr.v1.MsgToggleExchangeRateState",
  toAmino: (m: MsgToggleExchangeRateState) => clean({
    authority: m.authority || undefined,
    id: u64ToStr(m.id),
    state: m.state ? true : undefined,
  }),
  fromAmino: (a: any): MsgToggleExchangeRateState =>
    MsgToggleExchangeRateState.fromPartial({
      authority: a.authority ?? "",
      id: strToU64(a.id) != null ? Number(strToU64(a.id)!.toString()) : 0,
      state: a.state ?? false,
    }),
};
