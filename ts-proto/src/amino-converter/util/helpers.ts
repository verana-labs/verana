'use client'

/**
 * Utility helpers for cleaning objects and converting common types used in
 * protobuf/Amino JSON payloads (e.g., Long/u64, optional u32 wrappers, and dates).
 */

/* eslint-disable @typescript-eslint/no-explicit-any */
import Long from 'long';
import type { OptionalUInt32 } from '../../codec/verana/cs/v1/tx';

/** Removes undefined fields from an object to keep payloads clean. */
export const clean = <T extends Record<string, any>>(o: T): T => {
  Object.keys(o).forEach((k) => o[k] === undefined && delete o[k]);
  return o;
};

/** Normalizes a 64-bit value into its string representation (or undefined). */
export const u64ToStr = (v?: Long | string | number | null) =>
  v != null ? Long.fromValue(v).toString() : undefined;

/** Normalizes a 64-bit value into its string representation, omitting zero. */
export const u64ToStrIfNonZero = (v?: Long | string | number | null) => {
  if (v == null) return undefined;
  const value = Long.fromValue(v);
  return value.isZero() ? undefined : value.toString();
};

/** Parses a string into a 64-bit Long value (or undefined). */
export const strToU64 = (s?: string | null) =>
  s != null ? Long.fromString(s) : undefined;

// 0 -> "0" (string), >0 -> number
/** Converts a 32-bit number to an Amino-friendly format, preserving zero values. */
export const u32ToAmino = (n?: number | null) =>
  n == null ? undefined : (((n >>> 0) === 0) ? 0 : (n >>> 0));

/** Builds an OptionalUInt32 wrapper from loosely-typed input when possible. */
export const pickOptionalUInt32 = (v: any): OptionalUInt32 | undefined => {
  if (v === undefined || v === null) return undefined;
  if (typeof v === "string" && v.trim() === "") return undefined;
  const n = Number(v);
  if (!Number.isFinite(n)) return undefined;
  const value = (n >>> 0);
  return { value };
};

// 0 => {} (omitempty chain), >0 => {value:n}
/** Encodes an OptionalUInt32 wrapper into an Amino-style optional object. */
export const toOptU32Amino = (m?: { value: number } | undefined) => {
  if (!m) return undefined;
  const value = (Number(m.value) >>> 0);
  return value === 0 ? {} : { value };
};

// {}  (=> 0), {value:n}
/** Decodes an Amino-style optional object into an OptionalUInt32 wrapper. */
export const fromOptU32Amino = (x: any): OptionalUInt32 | undefined => {
  if (x == null) return undefined;
  // {} => wrapper, value default 0
  if (typeof x === "object" && x.value == null) return { value: 0 };

  const n = typeof x === "object" ? x.value : x;
  if (n === undefined || n === null) return undefined;
  if (typeof n === "string" && n.trim() === "") return undefined;

  const u = (Number(n) >>> 0);
  return { value: u };
};

/** Formats a Date into an ISO string commonly used in Amino JSON payloads. */
export const dateToIsoAmino = (d?: Date) => {
  if (!d) return undefined;
  return d.toISOString().replace(".000Z", "Z");
};

/** Parses an ISO date string into a Date instance (or undefined). */
export const isoToDate = (s?: string) => (s ? new Date(s) : undefined);
