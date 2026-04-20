import { formatUnits } from "viem";
import { useBookState } from "../hooks/useBookState";
import { useWalletBalances } from "../hooks/useWalletBalances";
import { PAIRS } from "../config/generated";
import { formatPrice } from "../lib/price";

interface Props {
  pair: string;
}

export function RecentTrades({ pair }: Props) {
  const { matches } = useBookState(pair);
  const { tokenInfo } = useWalletBalances();

  const pairConfig = PAIRS.find((p) => p.name === pair);
  const baseDecimals = pairConfig
    ? tokenInfo[pairConfig.baseToken.toLowerCase()]?.decimals
    : undefined;

  const formatQty = (raw: number) =>
    baseDecimals !== undefined
      ? formatUnits(BigInt(raw), baseDecimals)
      : raw.toString();

  const formatTime = (ts: number) => {
    const d = new Date(ts * 1000);
    return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" });
  };

  const recent = [...matches].reverse().slice(0, 20);

  if (recent.length === 0) {
    return (
      <div className="p-4 text-sm text-gray-500 text-center">
        No trades yet
      </div>
    );
  }

  return (
    <table className="w-full text-xs">
      <thead>
        <tr className="text-gray-500 border-b border-gray-800">
          <th className="text-right px-3 py-2 font-medium">Price</th>
          <th className="text-right px-3 py-2 font-medium">Quantity</th>
          <th className="text-right px-3 py-2 font-medium">Time</th>
        </tr>
      </thead>
      <tbody>
        {recent.map((trade, i) => (
          <tr key={i} className="border-b border-gray-800/50">
            <td className="px-3 py-2 text-right text-gray-300">
              {formatPrice(trade.price)}
            </td>
            <td className="px-3 py-2 text-right text-gray-300">
              {formatQty(trade.quantity)}
            </td>
            <td className="px-3 py-2 text-right text-gray-500">
              {formatTime(trade.timestamp)}
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
