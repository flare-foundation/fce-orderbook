import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAccount } from "wagmi";
import { placeOrder, type PlaceOrderReq, type PlaceOrderResp } from "../lib/orderbook";

export function usePlaceOrder() {
  const { address } = useAccount();
  const queryClient = useQueryClient();

  return useMutation<PlaceOrderResp, Error, Omit<PlaceOrderReq, "sender">>({
    mutationFn: (req) =>
      placeOrder({ ...req, sender: address!.toLowerCase() }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["myState"] });
      queryClient.invalidateQueries({ queryKey: ["bookState"] });
    },
  });
}
