import { useQuery } from "@tanstack/react-query";
import { getBookState, type BookStateResp, type PairState } from "../lib/orderbook";

/** Polls GET_BOOK_STATE every 2s. */
export function useBookState(pair: string) {
  const query = useQuery<BookStateResp>({
    queryKey: ["bookState"],
    queryFn: () => getBookState(),
    refetchInterval: 2000,
  });

  const pairState: PairState | undefined = query.data?.state?.pairs?.[pair];
  const allMatches = query.data?.state?.matches ?? [];
  const matches = allMatches.filter((m) => m.pair === pair);

  return {
    ...query,
    bids: pairState?.bids ?? [],
    asks: pairState?.asks ?? [],
    matches,
  };
}
