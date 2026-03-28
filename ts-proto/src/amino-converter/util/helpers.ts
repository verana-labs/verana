import Long from "long";
import type { Duration } from "../../codec/google/protobuf/duration";
import type { OptionalUInt32 } from "../../codec/verana/cs/v1/tx";

export const clean = <T extends Record<string, any>>(o: T): T => {
  Object.keys(o).forEach((k) => o[k] === undefined && delete o[k]);
  return o;
};

export const u64ToStr = (v?: Long | string | number | null) =>
  v != null ? Long.fromValue(v).toString() : undefined;

export const u64ToStrIfNonZero = (v?: Long | string | number | null) => {
  if (v == null) return undefined;
  const value = Long.fromValue(v);
  return value.isZero() ? undefined : value.toString();
};

export const strToU64 = (s?: string | null) =>
  s != null ? Long.fromString(s) : undefined;

export const u32ToAmino = (n?: number | null) =>
  n == null ? undefined : (((n >>> 0) === 0) ? 0 : (n >>> 0));

export const pickOptionalUInt32 = (v: any): OptionalUInt32 | undefined => {
  if (v === undefined || v === null) return undefined;
  if (typeof v === "string" && v.trim() === "") return undefined;
  const n = Number(v);
  if (!Number.isFinite(n)) return undefined;
  return { value: n >>> 0 };
};

export const pickU32 = pickOptionalUInt32;

export const toOptU32Amino = (m?: { value: number } | undefined) => {
  if (!m) return undefined;
  const value = Number(m.value) >>> 0;
  return value === 0 ? {} : { value };
};

export const fromOptU32Amino = (x: any): OptionalUInt32 | undefined => {
  if (x == null) return undefined;
  if (typeof x === "object" && x.value == null) return { value: 0 };

  const n = typeof x === "object" ? x.value : x;
  if (n === undefined || n === null) return undefined;
  if (typeof n === "string" && n.trim() === "") return undefined;

  return { value: Number(n) >>> 0 };
};

export const dateToIsoAmino = (d?: Date | null) => {
  if (d == null) return undefined;
  return d
    .toISOString()
    .replace(/\.000Z$/, "Z")
    .replace(/(\.\d*?[1-9])0+Z$/, "$1Z");
};

export const isoToDate = (s?: string | null) =>
  s != null ? new Date(s) : undefined;

export const dateToAmino = dateToIsoAmino;
export const dateFromAmino = isoToDate;

export const durationToAmino = (d?: Duration | null) => {
  if (!d) return undefined;
  return clean({
    seconds: Long.fromValue(d.seconds).toString(),
    nanos: d.nanos === 0 ? undefined : d.nanos,
  });
};

export const aminoToDuration = (
  d?: { seconds?: string | number | null; nanos?: string | number | null } | null,
): Duration | undefined => {
  if (!d) return undefined;
  return {
    seconds: d.seconds == null ? 0 : Number(d.seconds),
    nanos: d.nanos == null ? 0 : Number(d.nanos),
  };
};
