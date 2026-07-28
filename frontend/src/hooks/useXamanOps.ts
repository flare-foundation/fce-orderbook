// useXamanOps — the Xaman-wallet twins of useFaucet / useDeposit / useWithdraw.
//
// A PersonalAccount is gasless, so every on-chain movement is either relayed
// (xaman-service pays gas) or driven by an FSA XRP payment (the 0xFF memo):
//   faucet   → /relay/mint            (TestToken.mint is public; relayer pays gas)
//   deposit  → FSA approve if needed  (one XRP payment; sets MaxUint256 allowance)
//              → /relay/deposit       (depositFor pulls the PA's approved tokens)
//              → poll TEE result      (same threshold tag as the MetaMask path)
//   withdraw → WITHDRAW_REQUEST       (session key signs; TEE debits + signs slip)
//              → /relay/execute-withdraw
//
// The withdraw slip is persisted to localStorage between the TEE debit and the
// on-chain execute: it is the ONLY authorization to release the funds, and
// losing it after the debit would strand them. `retryExecute` resumes from a
// persisted slip (mirrors useWithdraw's cachedSignature pattern).

import { useEffect, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { erc20Abi } from "../abi/erc20";
import { maxUint256, type Address } from "viem";
import { encodeFunctionData } from "viem";
import { useXaman } from "../contexts/XamanContext";
import { readClient } from "../lib/readClient";
import { runFsaFlow, FSA_STAGES, type FsaStageId } from "../lib/fsaFlow";
import { relayDeposit, relayExecuteWithdraw, relayMint } from "../lib/relay";
import { requestWithdrawOffchain } from "../lib/fsaTee";
import { pollResult } from "../lib/teeClient";
import type { WithdrawResp } from "../lib/withdraw";
import { INSTRUCTION_SENDER } from "../config/addresses";
import { coston2 } from "../config/chain";
import type { StepReporter } from "../components/ui/ActionTray";

// ---- faucet ------------------------------------------------------------------

interface XamanFaucetArgs {
  token: Address;
  amount: bigint;
  report?: StepReporter;
}

export function useXamanFaucet() {
  const { pa } = useXaman();
  const queryClient = useQueryClient();

  return useMutation<string, Error, XamanFaucetArgs>({
    mutationFn: async ({ token, amount, report }) => {
      if (!pa) throw new Error("Xaman not connected");
      report?.detail("relaying mint");
      const { txHash } = await relayMint({ user: pa, token, amount });
      return txHash;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["readContracts"] });
    },
  });
}

// ---- deposit -----------------------------------------------------------------

/** Tray step labels, in the order the deposit mutation advances through them. */
export function xamanDepositSteps(symbol: string): string[] {
  return [`Approve ${symbol} (XRP payment)`, "Relay deposit", "TEE confirmation"];
}

interface XamanDepositArgs {
  token: Address;
  amount: bigint;
  report?: StepReporter;
}

export function useXamanDeposit() {
  const { pa, xrplAddress, setXamanRef } = useXaman();
  const queryClient = useQueryClient();

  return useMutation<string, Error, XamanDepositArgs>({
    mutationFn: async ({ token, amount, report }) => {
      if (!pa || !xrplAddress) throw new Error("Xaman not connected");
      const client = readClient();

      // Step 1: standing approval. One FSA payment sets MaxUint256, so every
      // later deposit skips straight to the relay.
      const allowance = await client.readContract({
        address: token,
        abi: erc20Abi,
        functionName: "allowance",
        args: [pa, INSTRUCTION_SENDER],
      });
      if (allowance < amount) {
        const stageLabel = (id: FsaStageId) => FSA_STAGES.find((s) => s.id === id)?.label ?? id;
        await runFsaFlow({
          account: xrplAddress,
          pa,
          calls: [{
            target: token,
            value: 0n,
            data: encodeFunctionData({
              abi: erc20Abi,
              functionName: "approve",
              args: [INSTRUCTION_SENDER, maxUint256],
            }),
          }],
          signHint: "approve the exchange to pull this token",
          onStage: (id, status, detail) =>
            report?.detail(`${stageLabel(id)}${status === "active" && detail ? ` — ${detail}` : status === "error" ? " — failed" : ""}`),
          onXamanRef: setXamanRef,
        });
      } else {
        report?.detail("already approved — skipped");
      }
      report?.advance();

      // Step 2: the relayer pulls the PA's approved tokens into the vault,
      // credited to the PA (depositFor).
      report?.detail("relayer submitting depositFor");
      const { txHash, instructionId } = await relayDeposit({ user: pa, token, amount });
      report?.advance();

      // Step 3: wait for the TEE to actually credit the deposit.
      if (!instructionId) {
        throw new Error("deposit relayed but no instruction id found — cannot confirm TEE processing");
      }
      const actionResult = await pollResult(
        instructionId, 30, 2000, "threshold",
        (n, max) => report?.detail(`attempt ${n}/${max}`),
      );
      if (actionResult.result.status === 0) {
        throw new Error(`TEE rejected deposit: ${actionResult.result.log}`);
      }
      if (actionResult.result.status === 2) {
        throw new Error(`TEE deposit still pending after polling (instruction ${instructionId})`);
      }
      return txHash;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["myState"] });
      queryClient.invalidateQueries({ queryKey: ["readContracts"] });
    },
  });
}

// ---- withdraw ------------------------------------------------------------------

/** Tray step labels for the Xaman withdraw path. */
export const XAMAN_WITHDRAW_STEPS = ["TEE authorization", "Relay execute"];

interface XamanWithdrawArgs {
  token: Address;
  amount: bigint;
  to: Address;
  report?: StepReporter;
}

const slipKey = (pa: string) => `fsa-withdraw-auth/${coston2.id}/${pa.toLowerCase()}`;

function loadSlip(pa: string): WithdrawResp | null {
  try {
    return JSON.parse(localStorage.getItem(slipKey(pa)) || "null");
  } catch {
    return null;
  }
}

export function useXamanWithdraw() {
  const { pa, session } = useXaman();
  const queryClient = useQueryClient();
  // Seed from localStorage so a slip stranded by a reload is still executable.
  const [pendingSlip, setPendingSlip] = useState<WithdrawResp | null>(() => (pa ? loadSlip(pa) : null));
  useEffect(() => {
    setPendingSlip(pa ? loadSlip(pa) : null);
  }, [pa]);

  function storeSlip(slip: WithdrawResp | null) {
    setPendingSlip(slip);
    if (!pa) return;
    try {
      if (slip) localStorage.setItem(slipKey(pa), JSON.stringify(slip));
      else localStorage.removeItem(slipKey(pa));
    } catch { /* quota */ }
  }

  async function execute(slip: WithdrawResp, report?: StepReporter): Promise<string> {
    report?.detail("relayer submitting executeWithdrawal");
    const { txHash } = await relayExecuteWithdraw({
      token: slip.token,
      amount: slip.amount,
      to: slip.to,
      withdrawalId: slip.withdrawalId,
      signature: slip.signature,
    });
    storeSlip(null);
    return txHash;
  }

  const mutation = useMutation<string, Error, XamanWithdrawArgs>({
    mutationFn: async ({ token, amount, to, report }) => {
      if (!pa || !session) throw new Error("Xaman not connected");

      // Step 1: session key signs; TEE debits and returns the signed slip.
      report?.detail("session key signing withdraw request");
      const slip = await requestWithdrawOffchain({ session, user: pa, token, to, amount });
      // The balance is debited from here on — the slip is the only way to get
      // the funds out, so persist it before touching the network again.
      storeSlip(slip);
      report?.advance();

      // Step 2: relay the slip on-chain.
      return execute(slip, report);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["myState"] });
      queryClient.invalidateQueries({ queryKey: ["readContracts"] });
    },
  });

  /** Retry the on-chain execute alone, from the persisted slip. */
  const retryExecute = async (report?: StepReporter) => {
    const slip = pendingSlip ?? (pa ? loadSlip(pa) : null);
    if (!slip) throw new Error("No pending withdrawal authorization");
    const tx = await execute(slip, report);
    queryClient.invalidateQueries({ queryKey: ["myState"] });
    queryClient.invalidateQueries({ queryKey: ["readContracts"] });
    return tx;
  };

  return { ...mutation, pendingSlip, retryExecute };
}
