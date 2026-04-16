import { ConnectButton } from "@rainbow-me/rainbowkit";
import { PairSelector } from "./PairSelector";
import { FaucetButton } from "./FaucetButton";
import { Button } from "./ui/Button";
import { env } from "../config/env";
import { PAIRS } from "../config/generated";

interface Props {
  selectedPair: string;
  onSelectPair: (pair: string) => void;
  onDeposit: () => void;
  onWithdraw: () => void;
}

export function Header({ selectedPair, onSelectPair, onDeposit, onWithdraw }: Props) {
  return (
    <header className="flex items-center justify-between px-4 py-3 border-b border-gray-800 bg-gray-900/80">
      <div className="flex items-center gap-4">
        <h1 className="text-lg font-bold text-white">Orderbook</h1>
        <PairSelector
          pairs={PAIRS.map((p) => p.name)}
          selected={selectedPair}
          onChange={onSelectPair}
        />
      </div>
      <div className="flex items-center gap-2">
        {env.showFaucet && <FaucetButton />}
        <Button variant="secondary" onClick={onDeposit}>
          Deposit
        </Button>
        <Button variant="secondary" onClick={onWithdraw}>
          Withdraw
        </Button>
        <ConnectButton showBalance={false} chainStatus="icon" />
      </div>
    </header>
  );
}
