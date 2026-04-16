import { useMyState } from "../hooks/useMyState";
import { useCancelOrder } from "../hooks/useCancelOrder";
import { Button } from "./ui/Button";
import { useToast } from "./ui/Toast";

export function OpenOrders() {
  const { openOrders } = useMyState();
  const cancelOrder = useCancelOrder();
  const { toast } = useToast();

  const handleCancel = async (orderId: string) => {
    try {
      await cancelOrder.mutateAsync({ orderId });
      toast("Order cancelled", "success");
    } catch (err) {
      toast(
        `Cancel failed: ${err instanceof Error ? err.message : "unknown"}`,
        "error"
      );
    }
  };

  if (openOrders.length === 0) {
    return (
      <div className="p-4 text-sm text-gray-500 text-center">
        No open orders
      </div>
    );
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-xs">
        <thead>
          <tr className="text-gray-500 border-b border-gray-800">
            <th className="text-left px-3 py-2 font-medium">Side</th>
            <th className="text-left px-3 py-2 font-medium">Pair</th>
            <th className="text-right px-3 py-2 font-medium">Price</th>
            <th className="text-right px-3 py-2 font-medium">Remaining</th>
            <th className="text-right px-3 py-2 font-medium"></th>
          </tr>
        </thead>
        <tbody>
          {openOrders.map((order) => (
            <tr key={order.id} className="border-b border-gray-800/50 hover:bg-gray-800/30">
              <td className={`px-3 py-2 ${order.side === "buy" ? "text-bid" : "text-ask"}`}>
                {order.side?.toUpperCase() ?? "?"}
              </td>
              <td className="px-3 py-2 text-gray-300">{order.pair ?? "-"}</td>
              <td className="px-3 py-2 text-right text-gray-300">{order.price ?? "-"}</td>
              <td className="px-3 py-2 text-right text-gray-300">{order.remaining}</td>
              <td className="px-3 py-2 text-right">
                <Button
                  variant="ghost"
                  className="text-xs px-2 py-1"
                  onClick={() => handleCancel(order.id)}
                  loading={cancelOrder.isPending}
                >
                  Cancel
                </Button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
