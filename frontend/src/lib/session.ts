// session.ts — the locally-held secp256k1 "session key" for the FSA auth model.
//
// In the FSA model the user's on-chain identity is a PersonalAccount (a contract, no
// private key), so a locally-generated session key signs the authenticated off-chain
// Direct ops (WITHDRAW_REQUEST). It is bound once to the PersonalAccount via
// BIND_SESSION_SIG; after that the XRP wallet (Xaman) is only needed to move funds.
//
// Signing uses the exact EIP-191 path the TEE expects (extension/crypto.go
// recoverSignerKey): personal_sign over keccak256(canonicalBytes).
//
// Key custody: derived from the Xaman SignIn signature whenever the wallet's signed
// blobs prove reproducible (see resolveSession) — then NOTHING is stored and any device
// recovers the key by just logging in. Until that proof, a localStorage backup keyed by
// (chainId, contract, xrplAddress) covers continuity; losing it costs one free re-bind.
//
// Ported to viem from shielded-transfer's ethers-based lib/session.js.

import { keccak256, sha256, type Address, type Hex } from "viem";
import { generatePrivateKey, privateKeyToAccount } from "viem/accounts";

export interface Session {
  privKey: Hex;
  /** EVM address of the session key (what the TEE recovers signatures to). */
  address: Address;
  /** 65-byte uncompressed pubkey (0x04 || X || Y) — what gets bound in the TEE. */
  sessionPubHex: Hex;
  /** EIP-191 personal_sign over keccak256(canonical) — matches recoverSignerKey. */
  sign(canonicalBytes: Hex): Promise<Hex>;
}

/** Build a session object from a 32-byte priv key. Pure — no storage. */
export function createSession(privKey: Hex): Session {
  const account = privateKeyToAccount(privKey);
  return {
    privKey,
    address: account.address,
    sessionPubHex: account.publicKey,
    sign: (canonicalBytes: Hex) =>
      account.signMessage({ message: { raw: keccak256(canonicalBytes) } }),
  };
}

/** Generate a fresh random session priv key. */
export function newSessionKey(): Hex {
  return generatePrivateKey();
}

/**
 * Derive a deterministic session priv key from a Xaman signed blob (response.hex).
 * XRPL secp256k1/Ed25519 signing is deterministic, so the same XRP-wallet approval of
 * the same (domain-separated) payload always yields the same key — nothing is stored,
 * and a new device just re-signs to get the same key back. sha256 the signed blob →
 * 32-byte priv (re-hash on the vanishingly-rare invalid-key miss). The blob embeds the
 * key-dependent TxnSignature, so only the wallet owner can reproduce it.
 */
export function deriveSessionPriv(signedHex: string): Hex {
  // Xaman returns the signed blob as raw hex WITHOUT a 0x prefix.
  const hex = (signedHex.startsWith("0x") ? signedHex : `0x${signedHex}`) as Hex;
  let priv = sha256(hex);
  for (let i = 0; i < 4; i++) {
    try {
      privateKeyToAccount(priv);
      return priv;
    } catch {
      priv = sha256(priv);
    }
  }
  throw new Error("session: failed to derive a valid secp256k1 key (vanishingly rare)");
}

export interface SessionScope {
  chainId: number;
  contract: string;
  xrplAddress: string;
}

function storageKey({ chainId, contract, xrplAddress }: SessionScope): string {
  return `fsa-session/${chainId}/${contract.toLowerCase()}/${xrplAddress.toLowerCase()}`;
}

/** Load the persisted session for (chainId, contract, xrplAddress), or null. */
export function loadSession(scope: SessionScope): Session | null {
  if (typeof localStorage === "undefined") return null;
  const hex = localStorage.getItem(storageKey(scope));
  return hex ? createSession(hex as Hex) : null;
}

export function persistSession(scope: SessionScope, session: Session): void {
  if (typeof localStorage !== "undefined") localStorage.setItem(storageKey(scope), session.privKey);
}

function clearSession(scope: SessionScope): void {
  if (typeof localStorage !== "undefined") localStorage.removeItem(storageKey(scope));
}

/**
 * Pick the session key for a login. Preference order:
 *
 *  1. the key DERIVED from this login's signed blob — nothing stored anywhere.
 *     Proven per wallet by the derived key matching the TEE's existing binding;
 *     on that proof the localStorage backup is DELETED.
 *  2. the persisted localStorage key, when it (and not the derived one) matches
 *     the binding — wallets whose signed blobs aren't reproducible keep working.
 *  3. no binding match at all (fresh account, or a new device for a
 *     non-deterministic wallet): use the stored key if any, else the derived key,
 *     and keep a persisted backup until a later login proves determinism.
 */
export function resolveSession(
  scope: SessionScope,
  signedHex: string | null,
  boundPubHex: Hex | null,
): { session: Session; source: string } {
  let derived: Session | null = null;
  try {
    derived = signedHex ? createSession(deriveSessionPriv(signedHex)) : null;
  } catch {
    /* malformed hex */
  }
  const stored = loadSession(scope);
  const bound = boundPubHex?.toLowerCase() ?? null;

  if (derived && bound && derived.sessionPubHex.toLowerCase() === bound) {
    clearSession(scope); // determinism proven for this wallet — drop the backup
    return { session: derived, source: "derived" };
  }
  if (stored && bound && stored.sessionPubHex.toLowerCase() === bound) {
    return { session: stored, source: "stored" };
  }
  // No binding match. Prefer an existing STORED key over re-deriving: a stored
  // key may correspond to an in-flight bind and is unrecoverable if overwritten,
  // while the derived key can always be recomputed on a later login.
  const session = stored || derived || createSession(newSessionKey());
  if (!stored) persistSession(scope, session); // backup until determinism is proven
  return { session, source: stored ? "stored-unbound" : derived ? "derived-unbound" : "new" };
}
