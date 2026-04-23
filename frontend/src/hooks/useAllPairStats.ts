import { useQuery } from '@tanstack/react-query';
import { getBookState, type BookStateResp, type BookMatch } from '../lib/orderbook';
import { formatPrice } from '../lib/price';

export interface PairStats {
  lastPrice: number;
  change24hPct: number;
}

/** Reads the shared bookState cache and derives last price + 24h change per pair. */
export function useAllPairStats(): Record<string, PairStats> {
  const query = useQuery<BookStateResp>({
    queryKey: ['bookState'],
    queryFn: () => getBookState(),
    refetchInterval: 2000,
  });

  const matches = query.data?.state?.matches ?? [];
  const byPair: Record<string, BookMatch[]> = {};
  for (const m of matches) {
    (byPair[m.pair] ??= []).push(m);
  }

  const stats: Record<string, PairStats> = {};
  for (const [pair, ms] of Object.entries(byPair)) {
    if (ms.length === 0) continue;
    const last = formatPrice(ms[0].price);
    const ref = ms.length > 1 ? formatPrice(ms[ms.length - 1].price) : last;
    const change = ref > 0 ? ((last - ref) / ref) * 100 : 0;
    stats[pair] = { lastPrice: last, change24hPct: change };
  }
  return stats;
}
