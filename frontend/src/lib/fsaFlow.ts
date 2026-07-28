// fsaFlow.ts — the one generic FSA payment driver: run contract call(s) from
// the user's PersonalAccount by having Xaman sign an XRPL payment carrying a
// 0xFF memo. Used by the Xaman deposit path (token approve). Ported from
// shielded-transfer's live-proven sequence.
//
// Stages (reported via onStage(id, status, detail)):
//   build → sign → xrpl → fdc → flare

import type { Address, Hex } from "viem";
import {
  personalAccountNonce, encodeCustomInstructionMemo,
  directMintingPaymentAddress, waitForUserOperationExecuted, toMemoData,
  type FsaCall,
} from "./fsa";
import { createPayload, pollPayload, type XamanPayloadRef } from "./xaman";
import { readClient } from "./readClient";
import { coston2 } from "../config/chain";

// ---- Pending-operation guard -----------------------------------------------
// A signed direct-mint payment stays valid until Flare's executor runs it —
// which can lag when the executor stalls. Building ANOTHER payment meanwhile
// reuses the same nonce; when the executor catches up, one wins and the loser
// reverts on-chain, parking its XRP at the Core Vault (no self-serve recovery
// on the 0xFF rail). So: record every SIGNED payment per PA in localStorage,
// refuse new payments until the chain's nonce passes the recorded one. Entries
// are written only after a real signature — a rejected/expired Xaman request
// can never dead-lock the guard.

interface PendingFsaEntry {
  nonce: string;
  txid?: string;
  label: string;
  at: number;
}

const pendingKey = (pa: string) => `fsa-pending/${coston2.id}/${pa.toLowerCase()}`;

/** The recorded in-flight payment for a PA, or null. */
export function pendingFsaOp(pa: string): PendingFsaEntry | null {
  if (typeof localStorage === "undefined") return null;
  try {
    return JSON.parse(localStorage.getItem(pendingKey(pa)) || "null");
  } catch {
    return null;
  }
}
function setPending(pa: string, entry: PendingFsaEntry) {
  try { localStorage.setItem(pendingKey(pa), JSON.stringify(entry)); } catch { /* quota */ }
}
function clearPending(pa: string) {
  try { localStorage.removeItem(pendingKey(pa)); } catch { /* denied */ }
}

/**
 * Re-check a recorded pending payment against the chain: returns the entry if
 * it is still waiting to execute, or clears it and returns null once the PA's
 * nonce has passed it.
 */
export async function refreshPendingFsaOp(pa: Address): Promise<PendingFsaEntry | null> {
  const entry = pendingFsaOp(pa);
  if (!entry) return null;
  const nonce = await personalAccountNonce(pa, readClient());
  if (BigInt(entry.nonce) < nonce) {
    clearPending(pa);
    return null;
  }
  return entry;
}

export type FsaStageId = "build" | "sign" | "xrpl" | "fdc" | "flare";
export type FsaStageStatus = "idle" | "active" | "done" | "error";
export type OnFsaStage = (id: FsaStageId, status: FsaStageStatus, detail?: string) => void;

export const FSA_STAGES: { id: FsaStageId; label: string }[] = [
  { id: "build", label: "Build instruction & preflight" },
  { id: "sign", label: "Sign the payment in Xaman" },
  { id: "xrpl", label: "XRPL ledger settlement" },
  { id: "fdc", label: "FDC attestation (~1.5–3 min)" },
  { id: "flare", label: "Atomic mint + call on Flare" },
];

// Proven-working minimum for a memo-carrying direct-mint payment. The surplus
// above mint fees lands as FXRP in the PA.
export const MIN_PAYMENT_DROPS = 100_000n; // 0.1 XRP

/**
 * Drive one FSA operation end to end. Throws on any failure (after reporting
 * the failing stage via onStage(id, 'error')). Resolves to the Flare execution
 * tx hash.
 */
export async function runFsaFlow({
  account, pa, calls, drops = MIN_PAYMENT_DROPS, onStage, onXamanRef, signHint,
}: {
  account: string;
  pa: Address;
  calls: FsaCall[];
  drops?: bigint;
  onStage?: OnFsaStage;
  onXamanRef?: (ref: XamanPayloadRef | null) => void;
  signHint?: string;
}): Promise<{ txid: string; txHash: Hex }> {
  const client = readClient();
  const stage: OnFsaStage = (id, status, detail) => onStage?.(id, status, detail);
  let current: FsaStageId = "build";
  try {
    stage("build", "active", "resolving nonce and Core Vault address…");
    const nonce = await personalAccountNonce(pa, client);

    const pending = pendingFsaOp(pa);
    if (pending && BigInt(pending.nonce) >= nonce) {
      throw new Error(
        `A previous operation ("${pending.label}", signed ${new Date(pending.at).toLocaleTimeString()}` +
        `${pending.txid ? `, XRPL tx ${pending.txid.slice(0, 10)}…` : ""}) is still waiting to execute on Flare. ` +
        "Sending another payment now would race it for the same account nonce — the losing payment gets parked " +
        "at the Core Vault. Wait for it to finish, then try again.",
      );
    }
    if (pending) clearPending(pa); // the chain's nonce has passed it — stale

    // Preflight every inner call from the PA BEFORE any XRP moves: the on-chain
    // flow is atomic, so a revert (token not allowed, extension id unset, …)
    // would strand the payment at the Core Vault.
    for (const call of calls) {
      try {
        await client.call({ account: pa, to: call.target, data: call.data, value: call.value });
      } catch (e) {
        const reason = e instanceof Error ? e.message.split("\n")[0] : "unknown reason";
        throw new Error(`Preflight failed — this would revert on-chain (${reason}). Nothing was sent.`);
      }
    }

    const memo = encodeCustomInstructionMemo({ personalAccount: pa, nonce, calls });
    const memoBytes = (memo.length - 2) / 2;
    if (memoBytes > 1024) {
      throw new Error(`FSA memo is ${memoBytes} bytes — exceeds the 1024-byte XRPL memo cap; split the calls`);
    }
    const dest = await directMintingPaymentAddress(client);
    stage("build", "done", `nonce ${nonce} · memo ${memoBytes} of 1024 bytes · preflight OK`);

    current = "sign";
    stage("sign", "active", signHint || "sign the payment in Xaman");
    const r = await createPayload("/sign", {
      txjson: {
        TransactionType: "Payment",
        Account: account,
        Destination: dest,
        Amount: String(drops), // NO DestinationTag — a tag would reroute the mint
        Memos: [{ Memo: { MemoData: toMemoData(memo) } }],
      },
    });
    onXamanRef?.(r);
    let txid: string;
    try {
      txid = await pollPayload(r.uuid, (p) => p.txid);
    } finally {
      onXamanRef?.(null);
    }
    // From here a valid payment exists on XRPL — arm the pending guard until
    // the chain's nonce moves past it (cleared below on success).
    setPending(pa, { nonce: nonce.toString(), txid, label: signHint || "FSA payment", at: Date.now() });
    stage("sign", "done", "signed");

    current = "xrpl";
    stage("xrpl", "active", "ledger closing…");
    await new Promise((res) => setTimeout(res, 4000));
    stage("xrpl", "done", "settled on the XRP Ledger");

    current = "fdc";
    stage("fdc", "active", "Flare data providers attest the payment; typical total 1.5–3 min");
    const started = Date.now();
    const txHash = await waitForUserOperationExecuted({ personalAccount: pa, nonce, client });
    clearPending(pa); // executed — the nonce is consumed, no race possible anymore
    stage("fdc", "done", `attested + executed after ${Math.round((Date.now() - started) / 1000)}s`);

    current = "flare";
    stage("flare", "done", "one atomic tx: mint FXRP → PersonalAccount → inner call");
    return { txid, txHash };
  } catch (e) {
    stage(current, "error", e instanceof Error ? e.message : String(e));
    throw e;
  }
}
