# Testing the orderbook against the live Coston2 deployment

The extension is deployed on Coston2 — TEE node and proxy are live on GCP. You run
only the frontend locally and trade against it. This gets you testing in ~10 minutes.

## What you need

**As a tester** (someone else runs the deployment):

- The repo, plus **Node.js + npm**. No Go, no Docker, no foundry, no GCP access.
- A browser wallet (MetaMask or similar) on **Coston2** (chain id `114`), with a
  little **C2FLR** from the [Flare faucet](https://faucet.flare.network/). Test
  tokens come from the app's own faucet button.
- Two things only the deployment owner (Jakob) can give you:
  - `DIRECT_API_KEY` — the proxy rejects every action without it
  - the current `EXTENSION_ID` + `INSTRUCTION_SENDER` (they're not in git)
- **For the Xaman/XRP flow** (in addition to MetaMask): the
  [Xaman](https://xaman.app) mobile app on **XRPL Testnet** (Settings → Advanced →
  Network settings → XRPL Testnet), a little testnet XRP from Xaman's built-in
  faucet, your own Xaman developer app credentials, your own C2FLR-funded relayer
  key, and `xaman-service` running locally (below).
- **You do not need** the deployment owner's deployer key, and you must not run
  anything in `scripts/` — see the warning at the bottom.

**To run the whole stack yourself** (deploy your own TEE, not needed for testing):
follow [deployment-steps.md](deployment-steps.md), including its **Known pitfalls**
section. It needs a funded deployer key and a Confidential Space VM. Don't point
`full-setup.sh` at a remote TEE — it writes one-shot on-chain state at the wrong
moment.

## Current deployment

| | |
|---|---|
| Extension ID | `0x000000000000000000000000000000000000000000000000000000000001014f` (65871) |
| InstructionSender | `0x007F55db2BA7c15a11F5Dd74b727223B5CD4d913` |
| Extension proxy | `https://tee-proxy-coston2-orderbook.flare.rocks` |
| Chain | Coston2, id `114`, RPC `https://coston2-api.flare.network/ext/C/rpc` |

Check it's up: `curl -s https://tee-proxy-coston2-orderbook.flare.rocks/info | head -c 200`

The two addresses change if the InstructionSender is redeployed — if you hit a
mismatch error below, confirm them with the deployment owner rather than assuming
this table is current.

## Setup

1. **`config/extension.env`** — not in git, create it. This is the only source of
   truth; `frontend/src/config/generated.ts` is a build artifact regenerated from
   it, and the copy committed on `main` is stale.

   ```bash
   cat > config/extension.env <<'EOF'
   EXTENSION_ID=0x000000000000000000000000000000000000000000000000000000000001014f
   INSTRUCTION_SENDER=0x007F55db2BA7c15a11F5Dd74b727223B5CD4d913
   EOF
   ```

2. **`frontend/.env`** — `cp .env.example .env`, then set:

   ```
   VITE_TEE_PROXY_URL=
   VITE_PROXY_UPSTREAM=https://tee-proxy-coston2-orderbook.flare.rocks
   VITE_DIRECT_API_KEY=<from the deployment owner>
   VITE_SHOW_FAUCET=true
   VITE_XAMAN_UPSTREAM=http://localhost:8787
   ```

   Leave `VITE_TEE_PROXY_URL` **empty** — vite then proxies `/direct`, `/state` and
   `/action` server-side, so the browser makes only same-origin requests and CORS
   never comes up. Setting it breaks that.

3. **Start it:** `cd frontend && npm install && npm run dev` → http://localhost:5173
   (vite falls back to `5174` if `5173` is taken — note which one it prints).

4. **Xaman flow only:**

   ```bash
   cd xaman-service && npm install
   cp .env.example .env      # then fill it in
   npm start                 # listens on :8787
   ```

   `.env` needs:

   - `XAMAN_API_KEY` / `XAMAN_API_SECRET` — from your own app in the
     [Xaman Developer Console](https://apps.xaman.dev). Set that app's allowed origin
     to whichever port vite printed. The secret stays server-side, never in the
     frontend.
   - `RELAYER_PRIVATE_KEY` — **your own** key, funded with C2FLR. It pays gas so XRP
     users need none. The service still starts without it, but logs
     `relay disabled` and the gasless deposit/withdraw endpoints won't work.
   - `CHAIN_RPC` — `https://coston2-api.flare.network/ext/C/rpc`

## Steps

1. **Connect** — _Connect Wallet_ in the header. With MetaMask, approve the Coston2
   network. With Xaman, scan the QR in the app; your XRPL address is mapped to a
   _PersonalAccount_, which becomes your identity in the exchange.
2. **Get test tokens** — in the wallet panel, use the faucet (shown while
   `VITE_SHOW_FAUCET=true`) to mint ~1000 of each pair token to your address.
3. **Deposit** — deposit into the exchange from the same panel. This is **two
   transactions**: an ERC-20 `approve`, then the `deposit` itself, so expect two
   wallet prompts. Balances then move from your wallet to **TEE AVAIL** (available to
   trade, held by the TEE).
4. **Place an order** — in **ORDER ENTRY**, pick `BUY`/`SELL` and `LIMIT`/`MARKET`,
   set `PRICE` (quote, USDT) and `SIZE` (base, FLR), then hit _BUY FLR_ / _SELL FLR_.
   `TOTAL` = price × size, so price is **per unit**, not the whole order.
5. **Watch the book** — your resting order appears in **ORDER BOOK**. Best bid is the
   highest price, best ask the lowest; both sit nearest the `SPREAD` line.
6. **Get a fill** — a fresh book has one side empty, so nothing matches. Place the
   opposite order at a crossing price; the trade shows in **TAPE · RECENT TRADES**
   and under your fills.
7. **Withdraw** — withdraw from the wallet panel; funds return to your wallet
   on-chain.

## If something's blocked

- **`401` on every action** → `VITE_DIRECT_API_KEY` doesn't match the deployed proxy.
- **Orders never fill** → no matching side on the book; place the opposite order.
- **`statement binds contract X, this TEE serves Y`** → your `config/extension.env`
  is out of date; ask the deployment owner for the current values.
- **`invalid TEE signature` on withdraw**, or **`INSTRUCTION_SENDER not configured on
  this TEE`**, or **`MAC resolver not configured`** → deployment-side, not yours.
  Tell the deployment owner.
- **Xaman QR does nothing** → `xaman-service` isn't running on 8787, or its
  credentials are missing.
- **Balances vanished** → they're in-memory; the TEE has no persistent volume, so a
  container restart clears balances and orders.
- **Empty ask side, `SPREAD —`** → normal on a fresh book, not a fault.

> [!WARNING]
> Don't run anything in `scripts/`. The TEE machine is already registered.
> `pre-build.sh`, `post-build.sh`, `extension-post-setup.sh` and `full-setup.sh`
> mutate the shared live deployment and write one-shot on-chain state that can't be
> undone — and they'd fail anyway without the deployment owner's deployer key.
