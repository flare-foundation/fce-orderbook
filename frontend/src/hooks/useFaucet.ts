import { useMutation } from "@tanstack/react-query";
import { useWriteContract } from "wagmi";
import type { Address } from "viem";
import { erc20Abi } from "../abi/erc20";

const FAUCET_AMOUNT = BigInt("1000000000000000000000"); // 1000e18

interface FaucetArgs {
  token: Address;
  to: Address;
}

export function useFaucet() {
  const { writeContractAsync } = useWriteContract();

  return useMutation<`0x${string}`, Error, FaucetArgs>({
    mutationFn: async ({ token, to }) => {
      const tx = await writeContractAsync({
        address: token,
        abi: erc20Abi,
        functionName: "mint",
        args: [to, FAUCET_AMOUNT],
      });
      return tx;
    },
  });
}
