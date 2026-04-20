import { useState } from "react";
import { useAccount } from "wagmi";
import { parseUnits, formatUnits } from "viem";
import { Dialog } from "./ui/Dialog";
import { Input } from "./ui/Input";
import { Button } from "./ui/Button";
import { useDeposit } from "../hooks/useDeposit";
import { useWalletBalances } from "../hooks/useWalletBalances";
import { useToast } from "./ui/Toast";
import { PAIRS, INSTRUCTION_SENDER } from "../config/generated";
import type { Address } from "viem";

interface Props {
  open: boolean;
  onClose: () => void;
}

export function DepositDialog({ open, onClose }: Props) {
  const { isConnected } = useAccount();
  const deposit = useDeposit();
  const { tokenInfo } = useWalletBalances();
  const { toast } = useToast();
  const [tokenIdx, setTokenIdx] = useState(0);
  const [amount, setAmount] = useState("");

  // Build token list from pairs config.
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

  const selected = tokens[tokenIdx];
  const info = selected ? tokenInfo[selected.address.toLowerCase()] : undefined;
  const decimals = info?.decimals;
  const walletFormatted =
    info?.balance !== undefined && decimals !== undefined
      ? formatUnits(info.balance, decimals)
      : "—";

  const handleDeposit = async () => {
    if (!selected) return;
    if (decimals === undefined) {
      toast("Loading token decimals, please try again", "error");
      return;
    }
    let scaled: bigint;
    try {
      scaled = parseUnits(amount, decimals);
    } catch {
      toast("Invalid amount", "error");
      return;
    }
    if (scaled <= 0n) {
      toast("Amount must be greater than 0", "error");
      return;
    }

    try {
      await deposit.mutateAsync({
        instructionSender: INSTRUCTION_SENDER as Address,
        token: selected.address,
        amount: scaled,
      });
      toast(`Deposited ${amount} ${selected.symbol}`, "success");
      setAmount("");
      onClose();
    } catch (err) {
      toast(
        `Deposit failed: ${err instanceof Error ? err.message : "unknown"}`,
        "error"
      );
    }
  };

  return (
    <Dialog open={open} onClose={onClose} title="Deposit">
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

        <div>
          <Input
            label={`Amount (${selected?.symbol ?? ""})`}
            type="number"
            min="0"
            step="any"
            placeholder="100"
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
          />
          <div className="text-xs text-gray-500 mt-1 flex justify-between">
            <span>
              Wallet: {walletFormatted} {selected?.symbol}
            </span>
            <button
              type="button"
              className="text-blue-400 hover:text-blue-300"
              onClick={() =>
                info?.balance !== undefined &&
                decimals !== undefined &&
                setAmount(formatUnits(info.balance, decimals))
              }
              disabled={info?.balance === undefined || decimals === undefined}
            >
              Max
            </button>
          </div>
        </div>

        <p className="text-xs text-gray-500">
          This will approve and deposit tokens to the orderbook vault.
          The TEE will credit your internal balance.
        </p>

        <Button
          onClick={handleDeposit}
          loading={deposit.isPending}
          disabled={!isConnected}
          className="w-full"
        >
          {deposit.isPending ? "Depositing..." : "Deposit"}
        </Button>
      </div>
    </Dialog>
  );
}
