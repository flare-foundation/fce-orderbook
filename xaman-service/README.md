# xaman-service

Minimal Node/Express backend for the FSA (Flare Smart Accounts) + Xaman wallet flow.
Adapted from `shielded-transfer/xaman-service`.

It exists because the Xaman (XUMM) API **secret must stay server-side**. The browser
never talks to Xaman directly — it asks this service to create sign payloads
(QR + deeplink) and polls their result. It also relays gas for gasless
XRPL-keyed PersonalAccounts.

## Endpoints

| Method | Path | Purpose |
|---|---|---|
| POST | `/login` | Create a SignIn payload (optional `memo`) → r-address + signed blob |
| POST | `/sign` | Create a sign payload for an arbitrary XRPL tx (the 0xFF FSA memo Payment) |
| GET | `/payload/:uuid` | Poll a payload → `{signed, resolved, account, txid, hex}` |
| POST | `/relay/mint` | Relayer pays gas → `TestToken.mint(user, amount)` (testnet faucet) |
| POST | `/relay/deposit` | Relayer pays gas + fee → `InstructionSender.depositFor(user, token, amount)`; returns `{txHash, instructionId}` |
| POST | `/relay/execute-withdraw` | Relayer submits the TEE's withdrawal slip → `InstructionSender.executeWithdrawal(...)` |

The InstructionSender address is read fresh per request from `../config/extension.env`
(written by the deploy scripts), so clients cannot aim the relayer at arbitrary contracts.

## Run

```bash
cp .env.example .env   # fill XAMAN_API_KEY / XAMAN_API_SECRET (+ RELAYER_PRIVATE_KEY for relays)
npm install
npm start              # listens on :8787
```

The frontend reaches it same-origin via Vite's dev proxy (`/login`, `/sign`,
`/payload`, `/relay` → `:8787`, see `frontend/vite.config.ts`). In production,
route it behind your reverse proxy.
