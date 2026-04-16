import { useMyState } from "../hooks/useMyState";
import { PAIRS } from "../config/generated";

/** Show per-token balances (available / held) from TEE state. */
export function Balances() {
  const { balances } = useMyState();

  // Build a token address -> symbol map from PAIRS config.
  const tokenNames: Record<string, string> = {};
  for (const pair of PAIRS) {
    const [base, quote] = pair.name.split("/");
    tokenNames[pair.baseToken.toLowerCase()] = base;
    tokenNames[pair.quoteToken.toLowerCase()] = quote;
  }

  const entries = Object.entries(balances);

  if (entries.length === 0) {
    return (
      <div className="p-4 text-sm text-gray-500 text-center">
        No balances — deposit tokens first
      </div>
    );
  }

  return (
    <table className="w-full text-xs">
      <thead>
        <tr className="text-gray-500 border-b border-gray-800">
          <th className="text-left px-3 py-2 font-medium">Token</th>
          <th className="text-right px-3 py-2 font-medium">Available</th>
          <th className="text-right px-3 py-2 font-medium">Held</th>
        </tr>
      </thead>
      <tbody>
        {entries.map(([addr, bal]) => (
          <tr key={addr} className="border-b border-gray-800/50">
            <td className="px-3 py-2 text-gray-300">
              {tokenNames[addr.toLowerCase()] ?? addr.slice(0, 10) + "..."}
            </td>
            <td className="px-3 py-2 text-right text-gray-100">
              {bal.available}
            </td>
            <td className="px-3 py-2 text-right text-gray-400">{bal.held}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
