import { useAccount } from "wagmi";
import { Button } from "./ui/Button";
import { useFaucet } from "../hooks/useFaucet";
import { useToast } from "./ui/Toast";
import { PAIRS } from "../config/generated";
import type { Address } from "viem";

export function FaucetButton() {
  const { address, isConnected } = useAccount();
  const faucet = useFaucet();
  const { toast } = useToast();

  const handleFaucet = async () => {
    if (!address) return;

    if (PAIRS.length === 0) {
      toast("No pairs configured", "error");
      return;
    }

    const tokens = new Set<Address>();
    for (const pair of PAIRS) {
      tokens.add(pair.baseToken as Address);
      tokens.add(pair.quoteToken as Address);
    }

    try {
      for (const token of tokens) {
        await faucet.mutateAsync({ token, to: address });
      }
      toast(`Minted 1000 of each test token (${tokens.size} tokens)`, "success");
    } catch (err) {
      toast(
        `Faucet failed: ${err instanceof Error ? err.message : "unknown"}`,
        "error"
      );
    }
  };

  return (
    <Button
      variant="ghost"
      onClick={handleFaucet}
      loading={faucet.isPending}
      disabled={!isConnected}
    >
      Faucet
    </Button>
  );
}
