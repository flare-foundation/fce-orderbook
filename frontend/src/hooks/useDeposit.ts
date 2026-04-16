import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useWriteContract } from "wagmi";
import { type Address } from "viem";
import { erc20Abi } from "../abi/erc20";
import { orderbookInstructionSenderAbi } from "../abi/orderbookInstructionSender";
import { INSTRUCTION_FEE } from "../lib/deposit";

interface DepositArgs {
  instructionSender: Address;
  token: Address;
  amount: bigint;
}

export function useDeposit() {
  const { writeContractAsync } = useWriteContract();
  const queryClient = useQueryClient();

  return useMutation<`0x${string}`, Error, DepositArgs>({
    mutationFn: async ({ instructionSender, token, amount }) => {
      // Step 1: Approve
      const approveTx = await writeContractAsync({
        address: token,
        abi: erc20Abi,
        functionName: "approve",
        args: [instructionSender, amount],
      });
      // Wait for approve to be mined.
      await waitForTx(approveTx);

      // Step 2: Deposit
      const depositTx = await writeContractAsync({
        address: instructionSender,
        abi: orderbookInstructionSenderAbi,
        functionName: "deposit",
        args: [token, amount],
        value: INSTRUCTION_FEE,
      });
      await waitForTx(depositTx);

      return depositTx;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["myState"] });
    },
  });
}

/** Simple wait for a tx to be mined. */
async function waitForTx(_hash: `0x${string}`): Promise<void> {
  // writeContractAsync already waits for the wallet to sign + submit.
  // We add a short delay to let the chain index the receipt.
  await new Promise((r) => setTimeout(r, 3000));
}
