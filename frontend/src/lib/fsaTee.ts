// fsaTee.ts — the TEE-side FSA direct ops: session-key binding and the
// off-chain withdraw request. Wire formats must match the Go handlers exactly
// (internal/extension/fsa_bind_sig.go, withdraw_request.go, pkg/types/types.go).

import { encodeAbiParameters, getAddress, parseAbiParameters, stringToHex, type Address, type Hex } from "viem";
import { sendDirectAndPoll } from "./teeClient";
import { createPayload, pollPayload, type XamanPayloadRef } from "./xaman";
import { INSTRUCTION_SENDER } from "../config/addresses";
import type { Session } from "./session";
import type { WithdrawResp } from "./withdraw";

// Must match internal/extension/fsa_bind_sig.go bindStatementDomain.
const BIND_SIG_DOMAIN = "Flare Orderbook v1";

// Must match pkg/types/types.go WithdrawRequestDomain.
const WITHDRAW_REQ_DOMAIN = stringToHex("FlareOrderbookWithdrawReqV1", { size: 32 });

/**
 * Millisecond-timestamp nonce: monotonic per user in practice (one human-driven
 * Xaman op per ms), and — unlike the ns nonces shielded-transfer uses — safely
 * below 2^53, so it survives JSON number round-trips into Go's uint64 exactly.
 */
export function freshNonce(): number {
  return Date.now();
}

export interface GetBindingResp {
  bound: boolean;
  target: string;
  sessionPub?: Hex;
  fingerprint?: string;
}

/** Unauthenticated lookup: is `target` bound, and to which session pubkey? */
export function checkBinding(target: Address): Promise<GetBindingResp> {
  return sendDirectAndPoll<GetBindingResp>("GET_BINDING", { target });
}

export interface BindSessionSigResp {
  user: Address;
  xrplAddress: string;
  sessionPub: Hex;
  fingerprint: string;
}

/** The human-readable statement the user signs in Xaman (as the SignIn memo). */
export function bindSigStatement(sessionPubHex: Hex, nonce: number): string {
  return `${BIND_SIG_DOMAIN} | bind session key ${sessionPubHex} | contract ${INSTRUCTION_SENDER} | nonce ${nonce}`;
}

/**
 * Bind `session` to the caller's PersonalAccount via an in-enclave-verified
 * Xaman signature: the user signs a SignIn whose memo is the binding statement,
 * and the TEE verifies the XRPL signature, resolves the PersonalAccount, and
 * binds the statement's session key. Free — no FXRP mint, no gas, no FDC wait.
 * `onXamanRef` receives the sign-request payload (QR) while the user approves,
 * then null.
 */
export async function bindSessionKeySig({
  session,
  onXamanRef,
}: {
  session: Session;
  onXamanRef?: (ref: XamanPayloadRef | null) => void;
}): Promise<BindSessionSigResp> {
  const statement = bindSigStatement(session.sessionPubHex, freshNonce());
  const r = await createPayload("/login", { memo: statement });
  onXamanRef?.(r);
  let hex: string;
  try {
    hex = await pollPayload(r.uuid, (p) => p.hex);
  } finally {
    onXamanRef?.(null);
  }
  return sendDirectAndPoll<BindSessionSigResp>("BIND_SESSION_SIG", {
    contract: INSTRUCTION_SENDER,
    xrplBlob: `0x${hex}`,
  });
}

/**
 * Canonical byte-string signed for an off-chain WITHDRAW_REQUEST. Layout:
 * abi.encode(domain, contract, user, token, to, amount, nonce) — must match
 * types.CanonicalWithdrawRequestBytes in Go.
 */
export function canonicalWithdrawRequestBytes({
  user, token, to, amount, nonce,
}: {
  user: Address; token: Address; to: Address; amount: bigint; nonce: number;
}): Hex {
  return encodeAbiParameters(
    parseAbiParameters("bytes32, address, address, address, address, uint256, uint256"),
    [WITHDRAW_REQ_DOMAIN, getAddress(INSTRUCTION_SENDER), getAddress(user), getAddress(token), getAddress(to), amount, BigInt(nonce)],
  );
}

/**
 * The off-chain twin of the on-chain withdraw: the session key signs the
 * canonical request, the TEE debits the balance and returns the same TEE-signed
 * withdrawal slip the on-chain path yields — in seconds, with no wallet tx.
 * The slip is then carried on-chain by the relayer (executeWithdrawal is
 * permissionless).
 */
export async function requestWithdrawOffchain({
  session, user, token, to, amount,
}: {
  session: Session; user: Address; token: Address; to: Address; amount: bigint;
}): Promise<WithdrawResp> {
  const nonce = freshNonce();
  const canonical = canonicalWithdrawRequestBytes({ user, token, to, amount, nonce });
  const signature = await session.sign(canonical);
  return sendDirectAndPoll<WithdrawResp>("WITHDRAW_REQUEST", {
    contract: INSTRUCTION_SENDER,
    user,
    token,
    to,
    amount: Number(amount),
    nonce,
    signature,
  });
}
