import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useAccount } from "wagmi";
import { getMyState, type GetMyStateResp } from "../lib/orderbook";

/** Polls GET_MY_STATE every 3s for the connected wallet. */
export function useMyState() {
  const { address } = useAccount();
  const queryClient = useQueryClient();

  const query = useQuery<GetMyStateResp>({
    queryKey: ["myState", address],
    queryFn: () => getMyState(address!.toLowerCase()),
    enabled: !!address,
    refetchInterval: 3000,
    retry: false,
  });

  const invalidate = () =>
    queryClient.invalidateQueries({ queryKey: ["myState", address] });

  return {
    ...query,
    balances: query.data?.balances ?? {},
    openOrders: query.data?.openOrders ?? [],
    invalidate,
  };
}
