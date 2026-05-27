/**
 * Manual protobuf encoder for cosmos.group.v1.ThresholdDecisionPolicy.
 *
 * The ts-proto codec for x/group is intentionally NOT generated in this repo
 * (the buf-pinned cosmos-sdk deps predate the x/group proto export — see
 * the comment on `verana.co.v1.Member` for context). We only need write-side
 * encoding for `MsgCreateCorporation.decision_policy`, so we hand-roll it
 * here using protobufjs/minimal (a transitive dep of ts-proto codecs).
 *
 * Wire format (cosmos-sdk proto):
 *
 *   message ThresholdDecisionPolicy {
 *     string threshold = 1;
 *     DecisionPolicyWindows windows = 2;
 *   }
 *
 *   message DecisionPolicyWindows {
 *     google.protobuf.Duration voting_period          = 1;
 *     google.protobuf.Duration min_execution_period   = 2;
 *   }
 *
 *   message google.protobuf.Duration {
 *     int64 seconds = 1;
 *     int32 nanos   = 2;
 *   }
 */

import * as _m0 from "protobufjs/minimal";
import Long = require("long");

export interface ThresholdDecisionPolicyInput {
  /** Decimal string, e.g. "1" or "0.5". MUST be non-empty. */
  threshold: string;
  /** Voting window in seconds. Defaults to 1 if omitted. */
  votingPeriodSeconds?: number;
  /** Min execution period in seconds. Defaults to 0 (no delay). */
  minExecutionPeriodSeconds?: number;
}

function encodeDuration(seconds: number): Uint8Array {
  const writer = _m0.Writer.create();
  // Field 1: int64 seconds
  if (seconds !== 0) {
    writer.uint32(8).int64(Long.fromNumber(seconds));
  }
  // Field 2: int32 nanos — always 0 here, omit when default.
  return writer.finish();
}

function encodeDecisionPolicyWindows(
  votingPeriodSeconds: number,
  minExecutionPeriodSeconds: number,
): Uint8Array {
  const writer = _m0.Writer.create();
  // Field 1: voting_period (Duration, length-delimited)
  const votingBytes = encodeDuration(votingPeriodSeconds);
  writer.uint32(10).bytes(votingBytes);
  // Field 2: min_execution_period (Duration). Always emit so the chain
  // canonicalises it consistently — encoders that omit zero-valued sub-
  // messages can confuse strict consumers.
  const minExecBytes = encodeDuration(minExecutionPeriodSeconds);
  writer.uint32(18).bytes(minExecBytes);
  return writer.finish();
}

/**
 * Encodes a ThresholdDecisionPolicy as raw protobuf bytes suitable for the
 * `value` field of a `google.protobuf.Any` wrapper. The Any's `type_url` MUST
 * be "/cosmos.group.v1.ThresholdDecisionPolicy".
 */
export function encodeThresholdDecisionPolicy(input: ThresholdDecisionPolicyInput): Uint8Array {
  const writer = _m0.Writer.create();
  if (!input.threshold) {
    throw new Error("ThresholdDecisionPolicy.threshold MUST be a non-empty decimal string");
  }
  // Field 1: threshold (string)
  writer.uint32(10).string(input.threshold);
  // Field 2: windows (DecisionPolicyWindows, length-delimited)
  const windowsBytes = encodeDecisionPolicyWindows(
    input.votingPeriodSeconds ?? 1,
    input.minExecutionPeriodSeconds ?? 0,
  );
  writer.uint32(18).bytes(windowsBytes);
  return writer.finish();
}
