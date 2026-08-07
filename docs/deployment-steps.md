# 🚀 TEE Extension Deployment — Step by Step

Linear recipe to deploy a TEE extension to Flare Coston or Coston2. Run the steps top to bottom.

## Prerequisites

- 🐳 Docker Desktop (Linux containers)
- 🐹 Go 1.25.1+
- 🔨 Foundry (`forge`, `cast`)
- `jq`
- Bash (Git Bash on Windows works)
- VPN access to Flare's indexer DB (`35.241.249.150:3306`)

## 1. Clone sibling repos

The extension's Dockerfiles consume both repos from `../../tee-node/`.

```text
<workspace>/tee/
├── tee-node/         # gitlab.com/flarenetwork/tee/tee-node, tag v0.0.20
├── tee-proxy/        # gitlab.com/flarenetwork/tee/tee-proxy, tag v0.0.17
└── extensions/
    └── <your-extension>/
```

## 2. Generate a funded deployer key

```bash
cast wallet new
cast wallet address --private-key 0x<private-key>
```

The derived address becomes your `INITIAL_OWNER`. Fund it from the target chain's faucet.

| Chain   | Faucet                                 |
| ------- | -------------------------------------- |
| Coston  | `https://faucet.flare.network/coston`  |
| Coston2 | `https://faucet.flare.network/coston2` |

## 3. Create `.env.<chain>`

Copy `.env.example` to `.env.coston` or `.env.coston2`. Fill in:

```bash
CHAIN=coston2                                                         # or coston
CHAIN_URL=https://coston2-api.flare.network/ext/C/rpc                 # chain RPC
ADDRESSES_FILE=./config/coston2/deployed-addresses.json
NORMAL_PROXY_URL=https://tee-proxy-coston2-1.flare.rocks              # FTDC proxy
EXT_PROXY_URL=                                                        # leave empty — set in Step 6

LOCAL_MODE=false
SIMULATED_TEE=false
DEPLOYMENT_PRIVATE_KEY=<private key, no 0x prefix>
INITIAL_OWNER=0x<derived address from Step 2>
```

Activate it:

```bash
bash ./scripts/use-chain.sh <chain>
```

Copies `.env.<chain>` → `.env`, which all scripts auto-load.

## 4. Register the extension on-chain

```bash
bash ./scripts/pre-build.sh
```

Compiles Solidity, deploys `InstructionSender`, registers the extension on-chain. Writes `EXTENSION_ID` and `INSTRUCTION_SENDER` to `config/extension.env`.

Read the new values — `EXTENSION_ID` is part of the hand-off in Step 6:

```bash
cat config/extension.env
```

## 5. Build the Docker image

Confirm `MODE=0` is the default in your extension's `Dockerfile` (`MODE=0` is the production attestation backend; `MODE=1` produces simulated attestation that FTDC rejects):

```dockerfile
ENV MODE=0 CONFIG_PORT=5501 SIGN_PORT=7701 EXTENSION_PORT=7702
```

Then build:

```powershell
$env:SOURCE_DATE_EPOCH = (git log -1 --format=%ct)
docker compose -f docker-compose.yaml build --no-cache extension-tee
docker tag <your-extension>-extension-tee:latest <your-extension>:v0.1.0
docker save <your-extension>:v0.1.0 -o <your-extension>-v0.1.0.tar
```

Setting `SOURCE_DATE_EPOCH` makes the build reproducible (same source → same `codeHash`).

Verify `MODE=0` is baked into the image:

```powershell
docker inspect <your-extension>:v0.1.0 --format '{{range .Config.Env}}{{println .}}{{end}}' | Select-String MODE
# expected: MODE=0
```

## 6. Deploy the image on a Confidential Space VM

Hand off (or deploy yourself) to a GCP Confidential Space VM with:

- The image — pass it **by digest** (`repo/extension-tee@sha256:…`), not by tag.
  Attestation pins the image's code hash and that hash is what you registered
  on-chain; a tag can be repointed, a digest cannot. If the image is mirrored into
  another registry, copy it by digest (`crane copy`) rather than rebuilding — a
  rebuild produces a different hash and attestation fails.
- Workload-launch env: `INITIAL_OWNER`, `CHAIN_URL`, `EXTENSION_ID` (from Step 4), `PROXY_URL` (proxy URL reachable from the TEE)
- Public HTTPS routed to port `6664` of the proxy container

You receive back the **public proxy URL**. Add it to `.env.<chain>` and re-activate:

```bash
# in .env.<chain>
EXT_PROXY_URL=<public proxy URL>
```

```bash
bash ./scripts/use-chain.sh <chain>
```

## 7. Verify the proxy `/info`

```powershell
curl -s $env:EXT_PROXY_URL/info | jq '.machineData'
```

Required values:

| Field          | Expected                                                          |
| -------------- | ----------------------------------------------------------------- |
| `platform`     | starts with `0x4743505f414d445f534556…` (GCP_AMD_SEV)             |
| `codeHash`     | real measured hash (**not** `0x194844cf…` — that's simulated)     |
| `extensionId`  | matches your `config/extension.env` `EXTENSION_ID`                |
| `initialOwner` | matches your `INITIAL_OWNER`                                      |

If `extensionId` is wrong, ask the VM operator to restart the container with the correct `EXTENSION_ID` env override (no image rebuild needed — it's a launch-policy override).

## 8. Register the TEE machine

> [!WARNING]
> Before running, ensure `scripts/post-build.sh` invokes `register-tee` with `-command rRap` (not the default `rap`):
>
> ```bash
> go run ./cmd/register-tee \
>     -a "$ADDRESSES_FILE" \
>     -c "$CHAIN_URL" \
>     -p "$EXT_PROXY_URL" \
>     -h "${EXT_PROXY_HOST_URL:-$EXT_PROXY_URL}" \
>     -ep "$NORMAL_PROXY_URL" \
>     -state "$PROJECT_DIR/config/register-tee.state" \
>     -command rRap \
>     || die "Register TEE failed"
> ```
>
> Step `a` (availability check) needs a one-time **challenge** — a random number from the contract that the TEE signs to prove it's alive. By default only `r` issues it, but `r` skips itself once the TEE is registered on-chain. So re-runs (image changes, diamond cuts, retries) revert with `Verification.ChallengeExpired`. Capital `R` issues the challenge directly — decoupled from `r` — so re-runs work.

Run:

```bash
bash ./scripts/post-build.sh
```

- `allow-tee-version` whitelists the codeHash for your extension.
- `register-tee -command rRap` pre-registers the TEE, requests fresh attestation, runs the FTDC availability check, promotes to production.

## 9. End-to-end test

```bash
bash ./scripts/test.sh
```

Sends test instructions through the deployed TEE and verifies the round-trip.

---

## When the extension image changes

1. Rebuild and hand off the new image.
2. The VM is re-deployed → `codeHash` changes.
3. `bash ./scripts/post-build.sh` whitelists the new codeHash.
4. `bash ./scripts/test.sh`.

## When the `FlareTeeManager` diamond is re-deployed

All extension registrations on that chain are wiped:

1. `bash ./scripts/pre-build.sh` — mints a fresh `EXTENSION_ID`.
2. Send the new `EXTENSION_ID` to the VM operator. They restart the container with `EXTENSION_ID=<new value>` as a launch-policy env override — no image rebuild needed.
3. Re-curl `/info` and confirm `extensionId` matches.
4. `bash ./scripts/post-build.sh`.
5. `bash ./scripts/test.sh`.

---

# Known pitfalls

Learned the hard way on the Coston2 deployment. Read this before deploying to a
remote TEE — several of these cost an InstructionSender each.

## The TEE key rotates on every container relaunch

Confidential Space has no persistent storage for the node, so each launch derives
a **new TEE keypair** → a new `teeId`. Consequences of every restart:

- The previously registered machine stays **active** on-chain with a key nobody
  holds. `getRandomTeeIds` load-balances across active machines, so instructions
  are routed to a dead node roughly half the time and silently never complete.
  **Pause the stale machine:**
  `cast send <FlareTeeManager> "pause(address)" <staleTeeId>` (owner-only; there is
  no `unpause`, only `toProduction` with a fresh availability proof).
- `teeAddress` on the InstructionSender no longer matches the signer — see below.

Corollary: **treat every relaunch as expensive** and batch config changes into one.

## `setTeeAddress` is one-shot — run `extension-post-setup.sh` LAST

`InstructionSender.sol` has `require(!teeAddressSet, "TEE address already set")`.
There is no setter and no reset. `executeWithdrawal` verifies the withdrawal
signature against `teeAddress`, so if it is locked to a stale key,
**withdrawals are permanently broken on that contract** and the only fix is
deploying a new InstructionSender.

Therefore:

- Run `extension-post-setup.sh` **only after the final relaunch**, and only when
  exactly one machine is active for the extension.
- **Never run `full-setup.sh` against a remote TEE** — it chains
  `extension-post-setup.sh` automatically and will lock `teeAddress` to whatever
  key exists at that moment, before the operator has relaunched.

The script now derives the live `teeId` from the proxy's `/info`
(`keccak256(pubkey.x ‖ pubkey.y)[12:]`) and refuses to run when more than one
machine is active — but it cannot know a relaunch is coming. That part is on you.

## Replacing the InstructionSender without a new `EXTENSION_ID`

`pre-build.sh` deploys a contract *and* registers a new extension. To swap only
the contract, keeping the extension (and its governance, version allowlist and
owner allowlist):

```bash
cd tools && go run ./cmd/deploy-contract -a ../config/coston2/deployed-addresses.json -c "$CHAIN_URL"
cast send <FlareTeeManager> "setExtensionContracts(uint256,address,address)" \
  <extensionId> 0x0000000000000000000000000000000000000000 <newInstructionSender> \
  --rpc-url "$CHAIN_URL" --chain 114 --private-key "$DEPLOYMENT_PRIVATE_KEY"
```

`onlyExtensionOwner`. Then update `INSTRUCTION_SENDER` in `config/extension.env`,
have the operator relaunch with it, and only then run `extension-post-setup.sh`.

## Launch env vars the operator must set

`INSTRUCTION_SENDER` and `CHAIN_URL` are easy to miss and fail in confusing ways:

| Missing | Symptom |
|---|---|
| `CHAIN_ID` | node leaves `chainID=0`; `SignResult` returns an **empty signature** instead of erroring, and the proxy panics with `signature must be 65 bytes, got 0` |
| `INSTRUCTION_SENDER` | `INSTRUCTION_SENDER not configured on this TEE` on `BIND_SESSION_SIG` |
| `CHAIN_URL` | `BIND_SESSION_SIG unavailable: MAC resolver not configured`. Distinct from `CHAIN_ID`; used for one read-only `getPersonalAccount` call on the MasterAccountController |
| `GOVERNANCE_SIGNERS`/`GOVERNANCE_THRESHOLD` | must match what `set-governance` put on-chain, or `register-tee` reverts. Both must be set together or the node errors |

Anything not listed in the image's `tee.launch_policy.allow_env_override` label is
**rejected outright** by the launcher (`env var ... is not allowed to be
overridden`) — adding a variable means a rebuild and a new tag.

`ADMIN_ADDRESSES` is deliberately **baked** rather than overridable: admins can read
other users' data via `EXPORT_HISTORY`, so it belongs in the code hash.

## `SIMULATED_TEE` must be `false` on real hardware

With `LOCAL_MODE=false` but `SIMULATED_TEE=true`, `register-tee` uses the hardcoded
test code hash `0x194844cf…` and fails with `code hashes do not match`. It would
otherwise register your machine with test attestation values.

## Rotating the proxy signing key

`PROXY_PRIVATE_KEY` belongs to the proxy VM, not the TEE, so rotating it does
**not** rotate the TEE key. But `teeProxyId` is bound into the machine record, so
after restarting the proxy, read the new proxyId from `/info` and call
`updateTeeMachineSettings(teeId, teeProxyId, url)`. Until then the availability
check sees a proxy signature that doesn't match the registration.

Never deploy with the Hardhat default key (`983760a4…` → `0xF4E02137…`); it is
public and is only appropriate for local runs.

## Balances are not persisted

`BALANCES_PATH` enables a balance snapshot (`pkg/balance/persist.go`) but is unset,
and Confidential Space launches with no mounts, so there is nowhere to write. A
container restart clears all balances and orders. Orders, book state and candles
have no persistence at all.

## Verifying a deployment is coherent

```bash
# node's view
curl -s "$EXT_PROXY_URL/info" | jq '.machineData | {extensionId, initialOwner, codeHash, governanceHash}'

# chain's view — these must agree with each other and with config/extension.env
cast call <FlareTeeManager> "getTeeExtensionInstructionsSender(uint256)(address)" <extensionId> --rpc-url "$CHAIN_URL" --chain 114
cast call <FlareTeeManager> "getActiveTeeMachines(uint256)(address[],string[])"   <extensionId> --rpc-url "$CHAIN_URL" --chain 114
cast call <InstructionSender> "teeAddress()(address)" --rpc-url "$CHAIN_URL" --chain 114
```

Green means: exactly **one** active machine, its `teeId` equals both the live node's
derived id and `teeAddress`, and the on-chain InstructionSender matches
`config/extension.env`.

`post-build.sh` exiting non-zero at `ToProduction` when the machine is already in
production is cosmetic — check `getTeeMachineStatus` (`2` = production) before
assuming anything failed.
