// readClient.ts — a standalone viem public client for lib code that runs
// outside React (FSA flows, session resolution). Components should keep using
// wagmi's usePublicClient; this exists so plain functions don't need a client
// threaded through every call.

import { createPublicClient, http, type PublicClient } from "viem";
import { coston2 } from "../config/chain";

let client: PublicClient | undefined;

export function readClient(): PublicClient {
  if (!client) {
    client = createPublicClient({ chain: coston2, transport: http() });
  }
  return client;
}
