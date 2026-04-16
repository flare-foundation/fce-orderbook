import { useState } from "react";
import { useAccount } from "wagmi";
import { Dialog } from "./ui/Dialog";
import { Input } from "./ui/Input";
import { Button } from "./ui/Button";
import { useWithdraw } from "../hooks/useWithdraw";
import { useToast } from "./ui/Toast";
import { PAIRS, INSTRUCTION_SENDER } from "../config/generated";
import type { Address } from "viem";

interface Props {
  open: boolean;
  onClose: () => void;
}

export function WithdrawDialog({ open, onClose }: Props) {
  const { address, isConnected } = useAccount();
  const withdraw = useWithdraw();
  const { toast } = useToast();
  const [tokenIdx, setTokenIdx] = useState(0);
  const [amount, setAmount] = useState("");

  const tokens: { symbol: string; address: Address }[] = [];
  for (const pair of PAIRS) {
    const [base, quote] = pair.name.split("/");
    if (!tokens.find((t) => t.address.toLowerCase() === pair.baseToken.toLowerCase())) {
      tokens.push({ symbol: base, address: pair.baseToken as Address });
    }
    if (!tokens.find((t) => t.address.toLowerCase() === pair.quoteToken.toLowerCase())) {
      tokens.push({ symbol: quote, address: pair.quoteToken as Address });
    }
  }

  const handleWithdraw = async () => {
    if (!address) return;
    const amt = Number(amount);
    if (amt <= 0) {
      toast("Amount must be greater than 0", "error");
      return;
    }

    try {
      await withdraw.mutateAsync({
        instructionSender: INSTRUCTION_SENDER as Address,
        token: tokens[tokenIdx].address,
        amount: BigInt(amt),
        to: address,
      });
      toast(`Withdrew ${amt} ${tokens[tokenIdx].symbol}`, "success");
      setAmount("");
      onClose();
    } catch (err) {
      toast(
        `Withdraw failed: ${err instanceof Error ? err.message : "unknown"}`,
        "error"
      );
    }
  };

  return (
    <Dialog open={open} onClose={onClose} title="Withdraw">
      <div className="flex flex-col gap-4">
        <div className="flex flex-col gap-1">
          <label className="text-xs text-gray-400 font-medium">Token</label>
          <select
            value={tokenIdx}
            onChange={(e) => setTokenIdx(Number(e.target.value))}
            className="bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-100"
          >
            {tokens.map((t, i) => (
              <option key={t.address} value={i}>
                {t.symbol}
              </option>
            ))}
          </select>
        </div>

        <Input
          label="Amount (raw units)"
          type="number"
          min="0"
          step="1"
          placeholder="100"
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
        />

        {withdraw.step && (
          <div className="text-xs text-blue-400 bg-blue-900/20 rounded-lg px-3 py-2">
            {withdraw.step}
          </div>
        )}

        {withdraw.cachedSignature && (
          <div className="text-xs text-yellow-400 bg-yellow-900/20 rounded-lg px-3 py-2">
            TEE signature cached — you can retry the on-chain execution.
            <Button
              variant="secondary"
              className="mt-2 w-full text-xs"
              onClick={() => withdraw.retryExecute(INSTRUCTION_SENDER as Address)}
            >
              Retry Execute
            </Button>
          </div>
        )}

        <p className="text-xs text-gray-500">
          Two-step process: sends a withdraw instruction, waits for TEE
          signature, then executes the withdrawal on-chain.
        </p>

        <Button
          onClick={handleWithdraw}
          loading={withdraw.isPending}
          disabled={!isConnected}
          className="w-full"
        >
          {withdraw.isPending ? "Withdrawing..." : "Withdraw"}
        </Button>
      </div>
    </Dialog>
  );
}
