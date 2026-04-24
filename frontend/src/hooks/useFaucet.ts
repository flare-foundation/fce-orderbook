import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useWriteContract, usePublicClient } from "wagmi";
import type { Address } from "viem";
import { erc20Abi } from "../abi/erc20";

const FAUCET_AMOUNT = BigInt("1000000000"); // 1000 tokens at 6 decimals (1000e6)

interface FaucetArgs {
  token: Address;
  to: Address;
}

export function useFaucet() {
  const { writeContractAsync } = useWriteContract();
  const publicClient = usePublicClient();
  const queryClient = useQueryClient();

  return useMutation<`0x${string}`, Error, FaucetArgs>({
    mutationFn: async ({ token, to }) => {
      const tx = await writeContractAsync({
        address: token,
        abi: erc20Abi,
        functionName: "mint",
        args: [to, FAUCET_AMOUNT],
      });
      // Wait for the mint to be mined before returning so the next refetch sees it.
      if (publicClient) {
        await publicClient.waitForTransactionReceipt({ hash: tx });
      }
      return tx;
    },
    onSuccess: () => {
      // Refresh wallet balances so minted tokens show up immediately.
      queryClient.invalidateQueries({ queryKey: ["readContracts"] });
    },
  });
}
