// fsa.ts — Flare Smart Accounts (XRPL) client helpers. Ported to viem from
// shielded-transfer's ethers-based lib/fsa.js; the logic and constants are identical.
//
// What this does: resolve an XRPL account's deterministic PersonalAccount on Flare, and
// build the `0xFF` custom-instruction memo that drives that PersonalAccount to call any
// contract as a side-effect of an FXRP direct-mint.
//
// Flow it feeds: the user signs ONE XRPL Payment (in Xaman) to the FAssets direct-minting
// address — directMintingPaymentAddress() below, NO destination tag — whose MemoData is
// the memo built here. Any indexer submits the FDC proof via
// AssetManagerFXRP.executeDirectMinting, which mints FXRP and calls back into
// MasterAccountController.handleMintedFAssets; that decodes memo[0]==0xFF and runs
// PackedUserOperation.callData = executeUserOp(Call[]) on the PersonalAccount. So the
// same XRP→FXRP mint that funds the account also drives the contract call — one
// signature, one bridge round-trip (FDC caps it at ~180 s).
//
// Constraints: XRPL caps a memo at 1024 bytes (header + abi.encode(userOp) must fit —
// split big batches across payments); the PA must hold native FLR for any payable call
// values. Docs: https://dev.flare.network/smart-accounts/memo-field-custom-instruction
//
// Memo layout — per dev.flare.network (10-byte header, then the ABI-encoded user op):
//
//   memo = uint8(0xFF) || uint8(walletId) || uint64(executorFeeUBA) || abi.encode(PackedUserOperation)
//
//   PackedUserOperation (OZ draft-IERC4337 v0.7 field order):
//     (address sender, uint256 nonce, bytes initCode, bytes callData,
//      bytes32 accountGasLimits, uint256 preVerificationGas, bytes32 gasFees,
//      bytes paymasterAndData, bytes signature)
//   callData = executeUserOp((address target, uint256 value, bytes data)[] calls)

import {
  encodeAbiParameters,
  encodeFunctionData,
  encodePacked,
  getAddress,
  hexToBytes,
  parseAbi,
  parseAbiItem,
  type Address,
  type Hex,
  type PublicClient,
} from "viem";

// MasterAccountController — same address on every Flare network (Coston2 + mainnet).
// In production resolve via the Flare contracts registry; pinned here for the demo.
export const MASTER_ACCOUNT_CONTROLLER: Address = "0x434936d47503353f06750Db1A444DBDC5F0AD37c";

// FlareContractRegistry — same address on every Flare network.
export const FLARE_CONTRACT_REGISTRY: Address = "0xaD67FE66660Fb8dFE9d6b1b4240d8650e30F6019";

// Memo instruction id for the memo-field custom instruction (full PackedUserOperation
// inline, dispatched to PersonalAccount.executeUserOp via handleMintedFAssets).
const FF_CUSTOM_INSTRUCTION = 0xff;

const ZERO_BYTES32 = ("0x" + "00".repeat(32)) as Hex;

const MAC_ABI = parseAbi([
  "function getPersonalAccount(string xrplOwner) view returns (address)",
  "function getNonce(address account) view returns (uint256)",
]);

const REGISTRY_ABI = parseAbi([
  "function getContractAddressByName(string) view returns (address)",
]);

const PA_ABI = parseAbi([
  "function executeUserOp((address target, uint256 value, bytes data)[] calls)",
]);

// PackedUserOperation tuple (OZ draft-IERC4337 v0.7). Order is load-bearing — it must
// match the on-chain abi.decode(_memoData[10:], (PackedUserOperation)).
const PACKED_USER_OP_TUPLE = {
  type: "tuple",
  components: [
    { name: "sender", type: "address" },
    { name: "nonce", type: "uint256" },
    { name: "initCode", type: "bytes" },
    { name: "callData", type: "bytes" },
    { name: "accountGasLimits", type: "bytes32" },
    { name: "preVerificationGas", type: "uint256" },
    { name: "gasFees", type: "bytes32" },
    { name: "paymasterAndData", type: "bytes" },
    { name: "signature", type: "bytes" },
  ],
} as const;

/** A single inner call the PersonalAccount executes. */
export interface FsaCall {
  target: Address;
  value: bigint;
  data: Hex;
}

/** XRPL MemoData convention: hex, NO 0x prefix, uppercase. */
export function toMemoData(hex: Hex): string {
  return Array.from(hexToBytes(hex))
    .map((x) => x.toString(16).padStart(2, "0"))
    .join("")
    .toUpperCase();
}

/**
 * Read the XRPL address (Core Vault) that direct-mint payments — including 0xFF custom
 * instructions — must be sent to. Resolved from the live AssetManagerFXRP via the
 * contracts registry, so it works unchanged on Coston2 and mainnet. The XRPL Payment
 * must NOT carry a destination tag (a tag reroutes the mint and drops the user op).
 */
export async function directMintingPaymentAddress(client: PublicClient): Promise<string> {
  const assetManager = await client.readContract({
    address: FLARE_CONTRACT_REGISTRY,
    abi: REGISTRY_ABI,
    functionName: "getContractAddressByName",
    args: ["AssetManagerFXRP"],
  });
  return client.readContract({
    address: assetManager,
    abi: parseAbi(["function directMintingPaymentAddress() view returns (string)"]),
    functionName: "directMintingPaymentAddress",
  });
}

/**
 * Resolve the deterministic PersonalAccount address for an XRPL r-address.
 * Works before the account is deployed (counterfactual CREATE2 address).
 */
export async function personalAccountFor(xrplAddress: string, client: PublicClient): Promise<Address> {
  const pa = await client.readContract({
    address: MASTER_ACCOUNT_CONTROLLER,
    abi: MAC_ABI,
    functionName: "getPersonalAccount",
    args: [xrplAddress],
  });
  return getAddress(pa);
}

/**
 * Read the PersonalAccount's current memo-instruction nonce. The UserOp.nonce in the
 * memo must equal this, or the on-chain run reverts (InvalidNonce).
 */
export async function personalAccountNonce(personalAccount: Address, client: PublicClient): Promise<bigint> {
  return client.readContract({
    address: MASTER_ACCOUNT_CONTROLLER,
    abi: MAC_ABI,
    functionName: "getNonce",
    args: [getAddress(personalAccount)],
  });
}

/**
 * Encode an FSA `0xFF` custom-instruction memo that runs `calls` on `personalAccount`.
 * Embed the returned bytes (via toMemoData) as the single MemoData of an XRPL Payment
 * to directMintingPaymentAddress(), no destination tag. Max 1024 bytes per memo.
 */
export function encodeCustomInstructionMemo({
  personalAccount,
  nonce,
  calls,
  walletId = 0,
  executorFee = 0n,
}: {
  personalAccount: Address;
  nonce: bigint;
  calls: FsaCall[];
  walletId?: number;
  executorFee?: bigint;
}): Hex {
  if (!Array.isArray(calls) || calls.length === 0) {
    throw new Error("encodeCustomInstructionMemo: calls must be a non-empty array");
  }
  const callData = encodeFunctionData({
    abi: PA_ABI,
    functionName: "executeUserOp",
    args: [calls.map((c) => ({ target: getAddress(c.target), value: c.value ?? 0n, data: c.data }))],
  });
  const encodedUserOp = encodeAbiParameters(
    [PACKED_USER_OP_TUPLE],
    [{
      sender: getAddress(personalAccount),
      nonce,
      initCode: "0x",
      callData,
      accountGasLimits: ZERO_BYTES32,
      preVerificationGas: 0n,
      gasFees: ZERO_BYTES32,
      paymasterAndData: "0x",
      signature: "0x",
    }],
  );
  return encodePacked(
    ["uint8", "uint8", "uint64", "bytes"],
    [FF_CUSTOM_INSTRUCTION, walletId, executorFee, encodedUserOp],
  );
}

const USER_OP_EXECUTED_EVENT = parseAbiItem(
  "event UserOperationExecuted(address indexed personalAccount, uint256 nonce)",
);

/**
 * Poll the MAC for UserOperationExecuted(personalAccount, nonce) — the proof that our
 * memo's user op ran on Flare. Chunked incremental scan because Coston2's public RPC
 * caps eth_getLogs at ~30 blocks. The FDC bridge round-trip is capped at ~180 s, hence
 * the 5 min default timeout. Resolves to the execution tx hash.
 */
export async function waitForUserOperationExecuted({
  personalAccount,
  nonce,
  client,
  initialLookback = 30n,
  chunk = 30n,
  timeoutMs = 300_000,
  intervalMs = 4000,
}: {
  personalAccount: Address;
  nonce: bigint;
  client: PublicClient;
  initialLookback?: bigint;
  chunk?: bigint;
  timeoutMs?: number;
  intervalMs?: number;
}): Promise<Hex> {
  const start = await client.getBlockNumber();
  let cursor = start > initialLookback ? start - initialLookback : 0n;
  const end = Date.now() + timeoutMs;
  for (;;) {
    const latest = await client.getBlockNumber();
    for (let lo = cursor; lo <= latest; lo += chunk) {
      const hi = lo + chunk - 1n > latest ? latest : lo + chunk - 1n;
      let logs;
      try {
        logs = await client.getLogs({
          address: MASTER_ACCOUNT_CONTROLLER,
          event: USER_OP_EXECUTED_EVENT,
          args: { personalAccount: getAddress(personalAccount) },
          fromBlock: lo,
          toBlock: hi,
        });
      } catch {
        continue;
      }
      const hit = logs.find((l) => l.args.nonce === nonce);
      if (hit) return hit.transactionHash;
      cursor = hi + 1n;
    }
    if (Date.now() >= end) break;
    await new Promise((r) => setTimeout(r, intervalMs));
  }
  throw new Error(
    `UserOperationExecuted(nonce ${nonce}) not seen within ${Math.round(timeoutMs / 1000)}s — ` +
    "the FDC attestation can take a few minutes; if the nonce advances later the op did run",
  );
}
