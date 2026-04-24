/**
 * orderbook.ts — typed wrappers for orderbook direct instructions.
 * Mirrors the request/response types from tools/cmd/test-orders/main.go.
 */

import { sendDirectAndPoll } from "./teeClient";

// --- Request types ---

export interface PlaceOrderReq {
  sender: string;
  pair: string;
  side: "buy" | "sell";
  type: "limit" | "market";
  price: number;
  quantity: number;
}

export interface CancelOrderReq {
  sender: string;
  orderId: string;
}

export interface GetMyStateReq {
  sender: string;
}

export interface GetBookStateReq {
  sender?: string;
}

// --- Response types ---

export interface Match {
  price: number;
  quantity: number;
}

export interface PlaceOrderResp {
  orderId: string;
  status: "resting" | "filled" | "partial";
  remaining: number;
  matches: Match[];
}

export interface CancelOrderResp {
  orderId: string;
  remaining: number;
}

export interface TokenBalance {
  available: number;
  held: number;
}

export interface OpenOrder {
  id: string;
  pair: string;
  side: "buy" | "sell";
  price: number;
  remaining: number;
  timestamp?: number;
}

export interface GetMyStateResp {
  balances: Record<string, TokenBalance>;
  openOrders: OpenOrder[];
  matches?: BookMatch[];
}

export interface PriceLevel {
  price: number;
  quantity: number;
}

export interface PairState {
  bids: PriceLevel[];
  asks: PriceLevel[];
}

export interface BookMatch {
  buyOrderId: string;
  sellOrderId: string;
  buyOwner: string;
  sellOwner: string;
  pair: string;
  price: number;
  quantity: number;
  timestamp: number;
}

export interface BookStateResp {
  state: {
    pairs: Record<string, PairState>;
    matchCount: number;
    matches: BookMatch[];
  };
}

// --- API wrappers ---

export function placeOrder(req: PlaceOrderReq): Promise<PlaceOrderResp> {
  return sendDirectAndPoll<PlaceOrderResp>("PLACE_ORDER", req);
}

export function cancelOrder(req: CancelOrderReq): Promise<CancelOrderResp> {
  return sendDirectAndPoll<CancelOrderResp>("CANCEL_ORDER", req);
}

export function getMyState(sender: string): Promise<GetMyStateResp> {
  return sendDirectAndPoll<GetMyStateResp>("GET_MY_STATE", { sender });
}

export function getBookState(sender?: string): Promise<BookStateResp> {
  return sendDirectAndPoll<BookStateResp>("GET_BOOK_STATE", { sender });
}
