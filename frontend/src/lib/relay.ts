// relay.ts — client for xaman-service's typed relay endpoints. A gasless
// account (an XRPL-keyed PersonalAccount) can't submit Flare transactions, so
// the service's funded key carries the safe-by-construction calls: mint (public
// testnet faucet), depositFor (user's own approved tokens → vault, credited to
// the same user) and executeWithdrawal (permissionless, pays exactly what the
// TEE's slip authorizes). Reached same-origin via the Vite dev proxy.

async function postRelay<T extends { txHash: string }>(path: string, body: unknown): Promise<T> {
  const r = await fetch(path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  const j = await r.json().catch(() => ({}));
  if (!r.ok) throw new Error(`${path}: ${j.error || `${r.status} ${r.statusText}`}`);
  if (!j.txHash) throw new Error(`${path}: missing txHash in response`);
  return j as T;
}

/** Relay TestToken.mint(user, amount) — testnet faucet for gasless accounts. */
export function relayMint({ user, token, amount }: { user: string; token: string; amount: bigint }) {
  return postRelay<{ txHash: string }>("/relay/mint", { user, token, amount: String(amount) });
}

/**
 * Relay InstructionSender.depositFor(user, token, amount). Resolves once mined,
 * with the DEPOSIT instruction id to poll the TEE result under (tag=threshold).
 */
export function relayDeposit({ user, token, amount }: { user: string; token: string; amount: bigint }) {
  return postRelay<{ txHash: string; instructionId: string | null }>("/relay/deposit", {
    user, token, amount: String(amount),
  });
}

/** Relay InstructionSender.executeWithdrawal with a TEE slip. Resolves once mined. */
export function relayExecuteWithdraw({
  token, amount, to, withdrawalId, signature,
}: {
  token: string; amount: number | bigint; to: string; withdrawalId: string; signature: string;
}) {
  return postRelay<{ txHash: string }>("/relay/execute-withdraw", {
    token, amount: String(amount), to, withdrawalId, signature,
  });
}
