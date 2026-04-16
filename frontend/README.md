# Orderbook Frontend

A classic orderbook trading UI for the Flare TEE orderbook extension. Connect a wallet, deposit, trade, and withdraw — all from a browser on Coston2 (chain ID 114).

## Quickstart (local dev)

Prerequisites: the orderbook extension must be running locally via Docker Compose (see the parent README).

```bash
# 1. Start the CORS proxy (from the orderbook root)
go run ./cmd/cors-proxy --target http://localhost:6664 --listen :6665 --allow-origin http://localhost:5173

# 2. Install dependencies and start the dev server
cd frontend
cp .env.example .env       # edit if your proxy URL differs
npm install
npm run dev                 # opens http://localhost:5173
```

The dev server proxies `/direct`, `/state`, and `/action` to the CORS proxy (`:6665`), which forwards to the TEE proxy (`:6664`).

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `VITE_TEE_PROXY_URL` | `http://localhost:6665` | TEE proxy URL (cors-proxy in dev) |
| `VITE_DIRECT_API_KEY` | `test-api-key-change-me` | API key for `/direct` endpoint |
| `VITE_SHOW_FAUCET` | `true` | Show the test-token faucet button |
| `VITE_WALLETCONNECT_PROJECT_ID` | (empty) | WalletConnect project ID |

## Config Sync

`npm run sync-config` reads deployed addresses and pair config from the parent repo and writes `src/config/generated.ts`. This runs automatically on `dev` and `build`.

The generated file includes:
- `EXTENSION_ID` — from `config/extension.env`
- `INSTRUCTION_SENDER` — the deployed contract address
- `BASE_TOKEN` / `QUOTE_TOKEN` — from `config/test-tokens.env`
- `PAIRS` — from `config/pairs.json`
- `INSTRUCTION_SENDER_ABI` — from the Foundry build output

## Production Build

```bash
npm run build    # output in dist/
npx serve dist   # verify the static bundle works
```

Set `VITE_TEE_PROXY_URL` to the production CORS proxy / API gateway URL before building.

## Extract as Standalone Repo

This frontend is designed to be self-contained. To extract it:

```bash
cp -r frontend ../my-orderbook-ui
cd ../my-orderbook-ui
rm -rf node_modules
git init
```

Then hand-edit `src/config/generated.ts` with your deployed addresses:

```ts
export const EXTENSION_ID = "your-extension-id";
export const INSTRUCTION_SENDER = "0xYourContractAddress";
export const BASE_TOKEN = "0xBaseTokenAddress";
export const QUOTE_TOKEN = "0xQuoteTokenAddress";
export const PAIRS = [{ name: "FLR/USDT", baseToken: "0x...", quoteToken: "0x..." }];
export const INSTRUCTION_SENDER_ABI = [ /* ... */ ];
```

Remove `scripts/sync-config.ts` and the `sync-config` npm script. The rest works standalone.
