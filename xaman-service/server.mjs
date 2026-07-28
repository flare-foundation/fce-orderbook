// xaman-service — minimal backend for the FSA "web dApp + backend" model.
//
// Holds the Xaman (XUMM) API key/secret (NEVER expose these to the browser) and turns
// transactions into Xaman sign payloads the frontend renders as a QR / push. Two flows:
//   - login: a SignIn payload  → returns the user's XRPL r-address once approved.
//   - sign:  a Payment (carrying the 0xFF FSA memo) → returns the signed txid.
//
// Run:
//   cp .env.example .env   # fill XAMAN_API_KEY / XAMAN_API_SECRET
//   npm install && npm start
//
// The frontend reaches this same-origin via Vite's dev proxy (/login, /sign, /payload,
// /relay — see frontend/vite.config.ts); in production, route it behind your reverse
// proxy instead.
//
// ⚠️ Written against the `xumm-sdk` v1 API. Method names (payload.create / payload.get) are
// stable, but verify the response shapes (next.always, refs.qr_png, meta.signed, response.account/txid)
// against your installed version — this backend is the one piece that cannot be tested without
// real Xaman credentials.

import express from 'express';
import cors from 'cors';
import { XummSdk } from 'xumm-sdk';

const { XAMAN_API_KEY, XAMAN_API_SECRET, PORT = 8787, CORS_ORIGIN = '*' } = process.env;
if (!XAMAN_API_KEY || !XAMAN_API_SECRET) {
  console.error('Set XAMAN_API_KEY and XAMAN_API_SECRET (see .env.example).');
  process.exit(1);
}

const xumm = new XummSdk(XAMAN_API_KEY, XAMAN_API_SECRET);
const app = express();
app.use(cors({ origin: CORS_ORIGIN }));
app.use(express.json({ limit: '256kb' }));

function payloadRefs(created) {
  return {
    uuid: created.uuid,
    deeplink: created.next?.always,
    qrPng: created.refs?.qr_png,
    websocket: created.refs?.websocket_status,
  };
}

// SignIn payload (login). Optional `memo` is a fixed domain string baked into the SignIn so
// the resulting signature is specific to this app; the signed blob (response.hex) is returned
// for callers that want to derive app-specific keys from it.
app.post('/login', async (req, res) => {
  try {
    const memo = req.body?.memo;
    const txjson = { TransactionType: 'SignIn' };
    if (memo && typeof memo === 'string') {
      txjson.Memos = [{ Memo: { MemoData: Buffer.from(memo, 'utf8').toString('hex').toUpperCase() } }];
    }
    const created = await xumm.payload.create({ txjson });
    res.json(payloadRefs(created));
  } catch (e) { res.status(500).json({ error: String(e?.message || e) }); }
});

// Sign request for an arbitrary XRPL tx (e.g. the FSA mint Payment carrying the 0xFF memo).
app.post('/sign', async (req, res) => {
  try {
    const txjson = req.body?.txjson;
    if (!txjson || typeof txjson !== 'object') return res.status(400).json({ error: 'missing txjson' });
    const created = await xumm.payload.create({ txjson });
    res.json(payloadRefs(created));
  } catch (e) { res.status(500).json({ error: String(e?.message || e) }); }
});

// Resolve a payload by uuid: signed?, the XRPL account (login), and the txid (sign).
app.get('/payload/:uuid', async (req, res) => {
  try {
    const p = await xumm.payload.get(req.params.uuid);
    res.json({
      signed: !!p?.meta?.signed,
      resolved: !!p?.meta?.resolved,
      account: p?.response?.account || null,
      txid: p?.response?.txid || null,
      hex: p?.response?.hex || null,
    });
  } catch (e) { res.status(500).json({ error: String(e?.message || e) }); }
});

// ---- Relayer -----------------------------------------------------------------
// Gasless accounts (XRPL-keyed PersonalAccounts) can't submit Flare transactions
// themselves. These STRICTLY-TYPED endpoints let this service pay the gas for the
// safe-by-construction calls of the fast path:
//   POST /relay/mint              {user, token, amount}  → TestToken.mint (public testnet faucet)
//   POST /relay/deposit           {user, token, amount}  → InstructionSender.depositFor
//   POST /relay/execute-withdraw  {token, amount, to, withdrawalId, signature}
//                                                        → InstructionSender.executeWithdrawal
// None of these can redirect value: depositFor only moves the user's own approved
// tokens INTO the vault credited to that same user, executeWithdrawal pays out
// exactly what the TEE's signature authorizes, to the destination it names, and
// mint targets the public-mint test tokens anyone can already mint. The worst a
// caller can do is spend this relayer's gas — acceptable for a testnet demo (a
// production deployment would add rate limiting / auth).
//
// The InstructionSender address is pinned SERVER-SIDE from ../config/extension.env
// (read fresh per request, so redeploys never go stale) — clients cannot aim the
// relayer at arbitrary contracts (mint excepted; testnet-only convenience).
// Requires RELAYER_PRIVATE_KEY + CHAIN_RPC.

import { readFileSync } from 'node:fs';
import { Contract, JsonRpcProvider, Wallet, getAddress } from 'ethers';

const { RELAYER_PRIVATE_KEY, CHAIN_RPC = 'https://coston2-api.flare.network/ext/C/rpc' } = process.env;
const INSTRUCTION_FEE_WEI = 1_000_000n; // matches frontend/src/lib/deposit.ts INSTRUCTION_FEE

function extensionEnv() {
  const raw = readFileSync(new URL('../config/extension.env', import.meta.url), 'utf8');
  return Object.fromEntries(raw.split(/\r?\n/)
    .filter((l) => l.includes('=') && !l.trim().startsWith('#'))
    .map((l) => [l.slice(0, l.indexOf('=')).trim(), l.slice(l.indexOf('=') + 1).trim()]));
}

function relayerWallet() {
  if (!RELAYER_PRIVATE_KEY) throw new Error('RELAYER_PRIVATE_KEY not set — relay endpoints disabled');
  return new Wallet(RELAYER_PRIVATE_KEY, new JsonRpcProvider(CHAIN_RPC));
}

// 2× gas buffer: Flare's eth_estimateGas underestimates calls that forward gas
// to sendInstructions (mirrors the frontend's deposit path).
async function sendWithBuffer(contract, method, args, overrides = {}) {
  const gasEst = await contract[method].estimateGas(...args, overrides);
  const tx = await contract[method](...args, { ...overrides, gasLimit: (gasEst * 2n) });
  const receipt = await tx.wait();
  if (receipt.status !== 1) throw new Error(`${method} reverted (hash=${receipt.hash})`);
  return receipt;
}

// The TeeExtensionRegistry emits TeeInstructionsSent(extensionId, instructionId,
// rewardEpochId, ...) — the only 4-topic log in a deposit tx. topic[2] is the
// instruction id the frontend polls the TEE result under (submissionTag=threshold).
// Mirrors frontend/src/lib/instructionId.ts.
function findInstructionId(logs) {
  for (const log of logs) {
    if (log.topics.length >= 4 && log.topics[2]) return log.topics[2];
  }
  return null;
}

// Testnet faucet relay: TestToken.mint is public; the relayer just pays the gas
// so XRPL-only users can obtain test tokens for their PersonalAccount.
app.post('/relay/mint', async (req, res) => {
  try {
    const { user, token, amount } = req.body || {};
    const erc20 = new Contract(
      getAddress(token),
      ['function mint(address to, uint256 amount)'],
      relayerWallet(),
    );
    const receipt = await sendWithBuffer(erc20, 'mint', [getAddress(user), BigInt(amount)]);
    res.json({ txHash: receipt.hash });
  } catch (e) { res.status(500).json({ error: String(e?.reason || e?.message || e) }); }
});

app.post('/relay/deposit', async (req, res) => {
  try {
    const { user, token, amount } = req.body || {};
    const env = extensionEnv();
    const sender = new Contract(
      env.INSTRUCTION_SENDER,
      ['function depositFor(address user, address token, uint256 amount) payable'],
      relayerWallet(),
    );
    const receipt = await sendWithBuffer(sender, 'depositFor',
      [getAddress(user), getAddress(token), BigInt(amount)],
      { value: INSTRUCTION_FEE_WEI });
    res.json({ txHash: receipt.hash, instructionId: findInstructionId(receipt.logs) });
  } catch (e) { res.status(500).json({ error: String(e?.reason || e?.message || e) }); }
});

app.post('/relay/execute-withdraw', async (req, res) => {
  try {
    const { token, amount, to, withdrawalId, signature } = req.body || {};
    const env = extensionEnv();
    const sender = new Contract(
      env.INSTRUCTION_SENDER,
      ['function executeWithdrawal(address token, uint256 amount, address to, bytes32 withdrawalId, bytes signature)'],
      relayerWallet(),
    );
    const receipt = await sendWithBuffer(sender, 'executeWithdrawal',
      [getAddress(token), BigInt(amount), getAddress(to), withdrawalId, signature]);
    res.json({ txHash: receipt.hash });
  } catch (e) { res.status(500).json({ error: String(e?.reason || e?.message || e) }); }
});

app.listen(Number(PORT), () => console.log(`xaman-service listening on :${PORT}${RELAYER_PRIVATE_KEY ? ' (relay enabled)' : ' (relay disabled — set RELAYER_PRIVATE_KEY)'}`));
