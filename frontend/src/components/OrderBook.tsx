import type { PriceLevel } from "../lib/orderbook";

interface Props {
  bids: PriceLevel[];
  asks: PriceLevel[];
  onPriceClick: (price: number) => void;
}

export function OrderBook({ bids, asks, onPriceClick }: Props) {
  // Show asks ascending (lowest at bottom, closest to spread).
  const sortedAsks = [...asks].sort((a, b) => b.price - a.price);
  const sortedBids = [...bids].sort((a, b) => b.price - a.price);

  const maxQty = Math.max(
    ...sortedAsks.map((a) => a.quantity),
    ...sortedBids.map((b) => b.quantity),
    1
  );

  return (
    <div className="flex flex-col h-full">
      <div className="grid grid-cols-2 text-xs text-gray-500 px-3 py-1 border-b border-gray-800">
        <span>Price</span>
        <span className="text-right">Quantity</span>
      </div>

      {/* Asks (top, red) */}
      <div className="flex-1 overflow-y-auto flex flex-col justify-end">
        {sortedAsks.map((level, i) => (
          <PriceLevelRow
            key={`ask-${i}`}
            level={level}
            side="ask"
            maxQty={maxQty}
            onClick={() => onPriceClick(level.price)}
          />
        ))}
      </div>

      {/* Spread indicator */}
      <div className="px-3 py-1 text-xs text-gray-500 border-y border-gray-800 text-center">
        {sortedAsks.length > 0 && sortedBids.length > 0
          ? `Spread: ${sortedAsks[sortedAsks.length - 1].price - sortedBids[0].price}`
          : "No spread"}
      </div>

      {/* Bids (bottom, green) */}
      <div className="flex-1 overflow-y-auto">
        {sortedBids.map((level, i) => (
          <PriceLevelRow
            key={`bid-${i}`}
            level={level}
            side="bid"
            maxQty={maxQty}
            onClick={() => onPriceClick(level.price)}
          />
        ))}
      </div>
    </div>
  );
}

function PriceLevelRow({
  level,
  side,
  maxQty,
  onClick,
}: {
  level: PriceLevel;
  side: "bid" | "ask";
  maxQty: number;
  onClick: () => void;
}) {
  const pct = (level.quantity / maxQty) * 100;
  const color = side === "bid" ? "text-bid" : "text-ask";
  const bg = side === "bid" ? "bg-green-500/10" : "bg-red-500/10";

  return (
    <div
      onClick={onClick}
      className="relative grid grid-cols-2 px-3 py-0.5 text-xs cursor-pointer hover:bg-gray-800/50"
    >
      <div
        className={`absolute inset-y-0 right-0 ${bg}`}
        style={{ width: `${pct}%` }}
      />
      <span className={`relative ${color}`}>{level.price}</span>
      <span className="relative text-right text-gray-300">
        {level.quantity}
      </span>
    </div>
  );
}
