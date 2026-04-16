import { useState } from "react";
import { Header } from "../components/Header";
import { OrderBook } from "../components/OrderBook";
import { OrderForm } from "../components/OrderForm";
import { OpenOrders } from "../components/OpenOrders";
import { Balances } from "../components/Balances";
import { RecentTrades } from "../components/RecentTrades";
import { DepositDialog } from "../components/DepositDialog";
import { WithdrawDialog } from "../components/WithdrawDialog";
import { Tabs } from "../components/ui/Tabs";
import { useBookState } from "../hooks/useBookState";
import { PAIRS } from "../config/generated";

export function Trade() {
  const firstPair = PAIRS[0];
  const defaultPair = firstPair ? firstPair.name : "FLR/USDT";
  const [pair, setPair] = useState(defaultPair);
  const [prefillPrice, setPrefillPrice] = useState<number | null>(null);
  const [depositOpen, setDepositOpen] = useState(false);
  const [withdrawOpen, setWithdrawOpen] = useState(false);
  const [rightTab, setRightTab] = useState("Balances");

  const { bids, asks } = useBookState(pair);

  return (
    <div className="flex flex-col h-screen">
      <Header
        selectedPair={pair}
        onSelectPair={setPair}
        onDeposit={() => setDepositOpen(true)}
        onWithdraw={() => setWithdrawOpen(true)}
      />

      <div className="flex-1 grid grid-cols-[1fr_320px_1fr] min-h-0">
        {/* Left: Orderbook */}
        <div className="border-r border-gray-800 flex flex-col min-h-0">
          <div className="px-3 py-2 text-xs font-medium text-gray-400 border-b border-gray-800">
            Order Book
          </div>
          <div className="flex-1 overflow-hidden">
            <OrderBook
              bids={bids}
              asks={asks}
              onPriceClick={(price) => setPrefillPrice(price)}
            />
          </div>
        </div>

        {/* Center: Order form */}
        <div className="border-r border-gray-800">
          <OrderForm pair={pair} prefillPrice={prefillPrice} />
        </div>

        {/* Right: Balances / Open Orders / Recent Trades */}
        <div className="flex flex-col min-h-0">
          <Tabs
            tabs={["Balances", "Open Orders", "Recent Trades"]}
            active={rightTab}
            onChange={setRightTab}
          />
          <div className="flex-1 overflow-y-auto">
            {rightTab === "Balances" && <Balances />}
            {rightTab === "Open Orders" && <OpenOrders />}
            {rightTab === "Recent Trades" && <RecentTrades pair={pair} />}
          </div>
        </div>
      </div>

      <DepositDialog open={depositOpen} onClose={() => setDepositOpen(false)} />
      <WithdrawDialog open={withdrawOpen} onClose={() => setWithdrawOpen(false)} />
    </div>
  );
}
