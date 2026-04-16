import { useState } from "react";
import { useAccount } from "wagmi";
import { Button } from "./ui/Button";
import { Input } from "./ui/Input";
import { Tabs } from "./ui/Tabs";
import { usePlaceOrder } from "../hooks/usePlaceOrder";
import { useToast } from "./ui/Toast";

interface Props {
  pair: string;
  prefillPrice: number | null;
}

export function OrderForm({ pair, prefillPrice }: Props) {
  const { isConnected } = useAccount();
  const [side, setSide] = useState<"buy" | "sell">("buy");
  const [orderType, setOrderType] = useState<"limit" | "market">("limit");
  const [price, setPrice] = useState(prefillPrice?.toString() ?? "");
  const [quantity, setQuantity] = useState("");
  const placeOrder = usePlaceOrder();
  const { toast } = useToast();

  // Update price when user clicks a price level in the orderbook.
  if (prefillPrice !== null && price !== prefillPrice.toString()) {
    setPrice(prefillPrice.toString());
  }

  const handleSubmit = async () => {
    const priceNum = orderType === "market" ? 0 : Number(price);
    const qtyNum = Number(quantity);

    if (qtyNum <= 0) {
      toast("Quantity must be greater than 0", "error");
      return;
    }
    if (orderType === "limit" && priceNum <= 0) {
      toast("Price must be greater than 0", "error");
      return;
    }

    try {
      const result = await placeOrder.mutateAsync({
        pair,
        side,
        type: orderType,
        price: priceNum,
        quantity: qtyNum,
      });
      toast(
        `Order ${result.status}: ${result.matches?.length ?? 0} fills`,
        result.status === "filled" ? "success" : "info"
      );
      setQuantity("");
    } catch (err) {
      toast(
        `Order failed: ${err instanceof Error ? err.message : "unknown error"}`,
        "error"
      );
    }
  };

  return (
    <div className="flex flex-col gap-3 p-4">
      <Tabs
        tabs={["buy", "sell"]}
        active={side}
        onChange={(t) => setSide(t as "buy" | "sell")}
      />

      <div className="flex gap-2">
        <button
          onClick={() => setOrderType("limit")}
          className={`text-xs px-3 py-1 rounded ${
            orderType === "limit"
              ? "bg-gray-700 text-white"
              : "text-gray-400 hover:text-gray-200"
          }`}
        >
          Limit
        </button>
        <button
          onClick={() => setOrderType("market")}
          className={`text-xs px-3 py-1 rounded ${
            orderType === "market"
              ? "bg-gray-700 text-white"
              : "text-gray-400 hover:text-gray-200"
          }`}
        >
          Market
        </button>
      </div>

      {orderType === "limit" && (
        <Input
          label="Price"
          type="number"
          min="0"
          step="any"
          placeholder="0"
          value={price}
          onChange={(e) => setPrice(e.target.value)}
        />
      )}

      <Input
        label="Quantity"
        type="number"
        min="0"
        step="any"
        placeholder="0"
        value={quantity}
        onChange={(e) => setQuantity(e.target.value)}
      />

      <Button
        variant={side === "buy" ? "primary" : "danger"}
        onClick={handleSubmit}
        loading={placeOrder.isPending}
        disabled={!isConnected}
        className={`w-full mt-2 ${
          side === "buy"
            ? "bg-bid hover:bg-green-700"
            : "bg-ask hover:bg-red-700"
        }`}
      >
        {!isConnected
          ? "Connect Wallet"
          : side === "buy"
            ? "Buy"
            : "Sell"}
      </Button>
    </div>
  );
}
