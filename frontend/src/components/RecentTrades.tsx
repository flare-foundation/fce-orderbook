import { useBookState } from "../hooks/useBookState";

interface Props {
  pair: string;
}

/**
 * Shows the tail of recent matches. Since the extension's GET_BOOK_STATE
 * returns the full state (not a match log), we show the current book
 * snapshot as a proxy. A proper trade history would require a dedicated
 * endpoint — this is an explicit non-goal per the plan.
 */
export function RecentTrades({ pair }: Props) {
  const { bids, asks } = useBookState(pair);

  // Combine and show as "recent activity" — best bid/ask levels.
  const levels = [
    ...asks.slice(0, 5).map((a) => ({ ...a, side: "sell" as const })),
    ...bids.slice(0, 5).map((b) => ({ ...b, side: "buy" as const })),
  ];

  if (levels.length === 0) {
    return (
      <div className="p-4 text-sm text-gray-500 text-center">
        No recent activity
      </div>
    );
  }

  return (
    <table className="w-full text-xs">
      <thead>
        <tr className="text-gray-500 border-b border-gray-800">
          <th className="text-left px-3 py-2 font-medium">Side</th>
          <th className="text-right px-3 py-2 font-medium">Price</th>
          <th className="text-right px-3 py-2 font-medium">Quantity</th>
        </tr>
      </thead>
      <tbody>
        {levels.map((level, i) => (
          <tr key={i} className="border-b border-gray-800/50">
            <td
              className={`px-3 py-2 ${
                level.side === "buy" ? "text-bid" : "text-ask"
              }`}
            >
              {level.side.toUpperCase()}
            </td>
            <td className="px-3 py-2 text-right text-gray-300">
              {level.price}
            </td>
            <td className="px-3 py-2 text-right text-gray-300">
              {level.quantity}
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
