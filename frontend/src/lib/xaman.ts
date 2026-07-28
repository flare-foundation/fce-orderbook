// xaman.ts — thin client for the local xaman-service (see ../../xaman-service/).
//
// The service holds the Xaman API secret and turns transactions into sign
// payloads (QR + deeplink). The browser reaches it same-origin through Vite's
// dev proxy (/login, /sign, /payload → :8787 — see vite.config.ts), so no CORS.
// Ported from shielded-transfer's proven driver.

/** What the service returns from POST /login and POST /sign. */
export interface XamanPayloadRef {
  uuid: string;
  deeplink?: string;
  qrPng?: string;
  websocket?: string;
}

/** The resolved payload state from GET /payload/:uuid. */
export interface XamanPayloadResult {
  signed: boolean;
  resolved: boolean;
  account: string | null;
  txid: string | null;
  hex: string | null;
}

/** POST /login or /sign — returns the payload refs (QR + deeplink). */
export async function createPayload(
  path: "/login" | "/sign",
  body?: unknown,
): Promise<XamanPayloadRef> {
  const r = await fetch(path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body || {}),
  });
  if (!r.ok) throw new Error(`${path}: ${r.status} ${await r.text().catch(() => "")}`);
  return r.json();
}

/**
 * Poll /payload/:uuid until the user resolves it in Xaman. `pick` extracts the
 * value we're waiting for — (p) => p.account for a login, (p) => p.txid for a
 * signed payment. Throws if the user rejects or the request times out.
 */
export async function pollPayload<T>(
  uuid: string,
  pick: (p: XamanPayloadResult) => T | null | undefined,
  { attempts = 120, intervalMs = 2500 } = {},
): Promise<T> {
  for (let i = 0; i < attempts; i++) {
    const r = await fetch(`/payload/${uuid}`);
    if (r.ok) {
      const p: XamanPayloadResult = await r.json();
      if (p.resolved || p.signed) {
        if (!p.signed) throw new Error("Rejected in Xaman");
        const v = pick(p);
        if (v) return v;
      }
    }
    await new Promise((res) => setTimeout(res, intervalMs));
  }
  throw new Error("Xaman request timed out");
}
