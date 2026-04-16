import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useWriteContract } from "wagmi";
import type { Address } from "viem";
import { orderbookInstructionSenderAbi } from "../abi/orderbookInstructionSender";
import { INSTRUCTION_FEE } from "../lib/deposit";
import { pollResult } from "../lib/teeClient";
import type { WithdrawResp } from "../lib/withdraw";

interface WithdrawArgs {
  instructionSender: Address;
  token: Address;
  amount: bigint;
  to: Address;
}

export function useWithdraw() {
  const { writeContractAsync } = useWriteContract();
  const queryClient = useQueryClient();
  const [step, setStep] = useState<string>("");
  const [cachedSignature, setCachedSignature] = useState<WithdrawResp | null>(null);

  const mutation = useMutation<`0x${string}`, Error, WithdrawArgs>({
    mutationFn: async ({ instructionSender, token, amount, to }) => {
      // Step 1: Send withdraw instruction on-chain.
      setStep("Sending withdraw transaction...");
      const withdrawTx = await writeContractAsync({
        address: instructionSender,
        abi: orderbookInstructionSenderAbi,
        functionName: "withdraw",
        args: [token, amount, to],
        value: INSTRUCTION_FEE,
      });

      // Wait a moment for the tx to be indexed.
      await new Promise((r) => setTimeout(r, 3000));

      // Step 2: Poll proxy for TEE-signed result.
      setStep("Waiting for TEE signature...");
      const actionResult = await pollResult(withdrawTx, 30, 2000);

      if (actionResult.result.status !== 1) {
        throw new Error(`Withdrawal failed: ${actionResult.result.log}`);
      }

      const wr: WithdrawResp = JSON.parse(actionResult.result.data);
      setCachedSignature(wr);

      // Step 3: Execute withdrawal on-chain.
      setStep("Executing withdrawal on-chain...");
      const executeTx = await writeContractAsync({
        address: instructionSender,
        abi: orderbookInstructionSenderAbi,
        functionName: "executeWithdrawal",
        args: [
          wr.token as Address,
          BigInt(wr.amount),
          wr.to as Address,
          wr.withdrawalId as `0x${string}`,
          wr.signature as `0x${string}`,
        ],
      });

      setStep("Done!");
      setCachedSignature(null);
      return executeTx;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["myState"] });
      setStep("");
    },
    onError: () => {
      setStep("");
    },
  });

  /** Retry step 3 if the user has a cached signature. */
  const retryExecute = async (instructionSender: Address) => {
    if (!cachedSignature) throw new Error("No cached signature");
    const wr = cachedSignature;

    setStep("Retrying executeWithdrawal...");
    const tx = await writeContractAsync({
      address: instructionSender,
      abi: orderbookInstructionSenderAbi,
      functionName: "executeWithdrawal",
      args: [
        wr.token as Address,
        BigInt(wr.amount),
        wr.to as Address,
        wr.withdrawalId as `0x${string}`,
        wr.signature as `0x${string}`,
      ],
    });
    setCachedSignature(null);
    setStep("");
    queryClient.invalidateQueries({ queryKey: ["myState"] });
    return tx;
  };

  return { ...mutation, step, cachedSignature, retryExecute };
}
