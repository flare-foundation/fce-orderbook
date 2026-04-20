import { formatUnits } from "viem";
import { useMyState } from "../hooks/useMyState";
import { useWalletBalances } from "../hooks/useWalletBalances";
import { PAIRS } from "../config/generated";

/** Show per-token balances: wallet (ERC20), available, and held (TEE state). */
export function Balances() {
  const { balances: teeBalances } = useMyState();
  const { tokenInfo } = useWalletBalances();

  const tokenNames: Record<string, string> = {};
  const allTokens = new Set<string>();
  for (const pair of PAIRS) {
    const [base, quote] = pair.name.split("/");
    const baseAddr = pair.baseToken.toLowerCase();
    const quoteAddr = pair.quoteToken.toLowerCase();
    tokenNames[baseAddr] = base;
    tokenNames[quoteAddr] = quote;
    allTokens.add(baseAddr);
    allTokens.add(quoteAddr);
  }

  // Normalize teeBalances keys to lowercase so lookups work regardless of
  // whether the TEE returns checksummed or lowercase addresses.
  const normalizedTeeBalances: typeof teeBalances = {};
  for (const [k, v] of Object.entries(teeBalances)) {
    normalizedTeeBalances[k.toLowerCase()] = v;
  }

  for (const addr of Object.keys(normalizedTeeBalances)) {
    allTokens.add(addr);
  }

  const rows = Array.from(allTokens).map((addr) => {
    const info = tokenInfo[addr];
    const decimals = info?.decimals;
    const tee = normalizedTeeBalances[addr];

    // If decimals haven't loaded yet, show raw numbers instead of silently
    // formatting with a wrong fallback (18) that makes small balances render as 0.
    const format = (raw: bigint) =>
      decimals === undefined ? raw.toString() : formatUnits(raw, decimals);

    return {
      addr,
      symbol: tokenNames[addr] ?? addr.slice(0, 10) + "...",
      wallet: info?.balance !== undefined ? format(info.balance) : "—",
      available: tee ? format(BigInt(tee.available)) : "0",
      held: tee ? format(BigInt(tee.held)) : "0",
    };
  });

  if (rows.length === 0) {
    return (
      <div className="p-4 text-sm text-gray-500 text-center">
        No tokens configured
      </div>
    );
  }

  return (
    <table className="w-full text-xs">
      <thead>
        <tr className="text-gray-500 border-b border-gray-800">
          <th className="text-left px-3 py-2 font-medium">Token</th>
          <th className="text-right px-3 py-2 font-medium">Wallet</th>
          <th className="text-right px-3 py-2 font-medium">Available</th>
          <th className="text-right px-3 py-2 font-medium">Held</th>
        </tr>
      </thead>
      <tbody>
        {rows.map((row) => (
          <tr key={row.addr} className="border-b border-gray-800/50">
            <td className="px-3 py-2 text-gray-300">{row.symbol}</td>
            <td className="px-3 py-2 text-right text-gray-100">{row.wallet}</td>
            <td className="px-3 py-2 text-right text-gray-100">
              {row.available}
            </td>
            <td className="px-3 py-2 text-right text-gray-400">{row.held}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
