import { useState, useEffect } from 'react';
import { parseUnits, formatUnits, type Address } from 'viem';
import { useIdentity } from '../hooks/useIdentity';
import { useDeposit, depositSteps } from '../hooks/useDeposit';
import { useWithdraw, WITHDRAW_STEPS } from '../hooks/useWithdraw';
import { useFaucet } from '../hooks/useFaucet';
import {
  useXamanFaucet, useXamanDeposit, useXamanWithdraw,
  xamanDepositSteps, XAMAN_WITHDRAW_STEPS,
} from '../hooks/useXamanOps';
import { useWalletBalances } from '../hooks/useWalletBalances';
import { useMyState } from '../hooks/useMyState';
import { useToast } from './ui/Toast';
import { useTray } from './ui/ActionTray';
import { PAIRS } from '../config/generated';
import { INSTRUCTION_SENDER } from '../config/addresses';

type Tab = 'FAUCET' | 'DEPOSIT' | 'WITHDRAW' | 'HISTORY';

const FAUCET_AMOUNTS: Record<string, string> = {
  FLR: '10000',
  USDT: '100000',
  BTC: '1',
  ETH: '15',
};
const DEFAULT_FAUCET_AMOUNT = '1000';

function faucetAmountFor(symbol: string): string {
  return FAUCET_AMOUNTS[symbol.toUpperCase()] ?? DEFAULT_FAUCET_AMOUNT;
}

interface HistoryEntry {
  id: string;
  type: string;
  symbol: string;
  amount: string;
  timestamp: number;
}

interface TokenEntry {
  address: Address;
  symbol: string;
}

// Get unique tokens from PAIRS config
function getTokens(): TokenEntry[] {
  const seen = new Set<string>();
  const tokens: TokenEntry[] = [];
  for (const p of PAIRS) {
    const [base, quote] = p.name.split('/');
    if (!seen.has(p.baseToken.toLowerCase())) {
      seen.add(p.baseToken.toLowerCase());
      tokens.push({ address: p.baseToken as Address, symbol: base });
    }
    if (!seen.has(p.quoteToken.toLowerCase())) {
      seen.add(p.quoteToken.toLowerCase());
      tokens.push({ address: p.quoteToken as Address, symbol: quote });
    }
  }
  return tokens;
}

function formatAddress(addr: string): string {
  if (!addr || addr.length < 12) return addr;
  return `${addr.slice(0, 6)}...${addr.slice(-4)}`.toLowerCase();
}

/**
 * Wallet/viem errors run to several paragraphs. Tray step details get one
 * short line — the first sentence is where the useful part lives.
 */
function shortReason(message: string): string {
  const first = message.split('\n')[0].trim();
  return first.length > 56 ? `${first.slice(0, 55)}…` : first;
}

interface WalletModalProps {
  open: boolean;
  onClose: () => void;
}

export function WalletModal({ open, onClose }: WalletModalProps) {
  const { address, walletKind } = useIdentity();
  const isXaman = walletKind === 'xaman';
  const [tab, setTab] = useState<Tab>('FAUCET');
  const [tokenIdx, setTokenIdx] = useState(0);
  const [amount, setAmount] = useState('');
  const [history, setHistory] = useState<HistoryEntry[]>([]);

  const { toast } = useToast();
  const tray = useTray();
  const deposit = useDeposit();
  const withdraw = useWithdraw();
  const faucet = useFaucet();
  const xamanFaucet = useXamanFaucet();
  const xamanDeposit = useXamanDeposit();
  const xamanWithdraw = useXamanWithdraw();
  const { tokenInfo } = useWalletBalances();
  const { balances: teeBalances } = useMyState();

  const depositPending = deposit.isPending || xamanDeposit.isPending;
  const withdrawPending = withdraw.isPending || xamanWithdraw.isPending;
  const faucetPending = faucet.isPending || xamanFaucet.isPending;

  const tokens = getTokens();

  // ESC closes modal
  useEffect(() => {
    if (!open) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [open, onClose]);

  // Reset form when tab/token changes
  useEffect(() => {
    setAmount('');
  }, [tab, tokenIdx]);

  if (!open) return null;

  const selectedToken = tokens[tokenIdx];
  const selectedInfo = selectedToken
    ? tokenInfo[selectedToken.address.toLowerCase()]
    : undefined;
  const decimals = selectedInfo?.decimals;

  function fmtWallet(addr: string): string {
    const info = tokenInfo[addr.toLowerCase()];
    if (info?.balance === undefined || info.decimals === undefined) return '—';
    return parseFloat(formatUnits(info.balance, info.decimals)).toFixed(4);
  }

  function fmtTee(addr: string, field: 'available' | 'held'): string {
    const bal = teeBalances[addr.toLowerCase()];
    const info = tokenInfo[addr.toLowerCase()];
    if (!bal || info?.decimals === undefined) return '—';
    try {
      return parseFloat(
        formatUnits(BigInt(Math.floor(bal[field])), info.decimals)
      ).toFixed(4);
    } catch {
      return '—';
    }
  }

  function addHistory(type: string, sym: string, amt: string) {
    setHistory(h => [
      {
        id: String(Date.now()),
        type,
        symbol: sym,
        amount: amt,
        timestamp: Date.now(),
      },
      ...h.slice(0, 49),
    ]);
  }

  // ---- FAUCET ----
  async function handleFaucet() {
    if (!address) {
      toast('Connect wallet first', 'error');
      return;
    }

    // Resolve the mintable set up front so the tray's steps line up 1:1 with
    // the mints we actually attempt.
    const mintable: { symbol: string; addr: Address; raw: bigint; human: string }[] = [];
    for (const t of getTokens()) {
      const info = tokenInfo[t.address.toLowerCase()];
      if (info?.decimals === undefined) continue;
      const human = faucetAmountFor(t.symbol);
      try {
        mintable.push({
          symbol: t.symbol,
          addr: t.address,
          raw: parseUnits(human, info.decimals),
          human,
        });
      } catch {
        // Unparseable amount for this token — skip it.
      }
    }
    if (!mintable.length) {
      toast('Token decimals not loaded yet — try again in a moment', 'error');
      return;
    }

    const job = tray.start({
      title: 'Mint test tokens',
      steps: mintable.map(m => `Mint ${m.symbol}`),
    });

    const failed: string[] = [];
    for (const [i, m] of mintable.entries()) {
      if (i > 0) job.advance();
      try {
        if (isXaman) {
          // Gasless PersonalAccount: the relayer pays the mint gas.
          await xamanFaucet.mutateAsync({ token: m.addr, amount: m.raw, report: job });
        } else {
          await faucet.mutateAsync({
            token: m.addr,
            to: address,
            amount: m.raw,
            report: job,
          });
        }
        addHistory('MINT', m.symbol, m.human);
      } catch (e) {
        // One token failing shouldn't abort the rest — mark it and carry on.
        failed.push(m.symbol);
        job.failStep(e instanceof Error ? shortReason(e.message) : 'failed');
      }
    }

    const minted = mintable.length - failed.length;
    if (minted === 0) {
      job.fail({ message: `All ${mintable.length} mints failed.` });
      toast('Faucet failed', 'error');
      return;
    }
    job.finish({ summary: { minted: `${minted}/${mintable.length}` } });
    toast(
      failed.length
        ? `Minted ${minted}/${mintable.length} — failed: ${failed.join(', ')}`
        : `Minted ${minted} test tokens`,
      failed.length ? 'info' : 'success',
    );
  }

  // ---- DEPOSIT ----
  async function handleDeposit() {
    if (!selectedToken) return;
    if (decimals === undefined) {
      toast('Loading token decimals, please try again', 'error');
      return;
    }
    if (!amount) return;
    let rawAmount: bigint;
    try {
      rawAmount = parseUnits(amount, decimals);
    } catch {
      toast('Invalid amount', 'error');
      return;
    }
    if (rawAmount <= 0n) {
      toast('Amount must be greater than 0', 'error');
      return;
    }
    const job = tray.start({
      title: `Deposit ${amount} ${selectedToken.symbol}`,
      steps: isXaman ? xamanDepositSteps(selectedToken.symbol) : depositSteps(selectedToken.symbol),
    });
    try {
      const tx = isXaman
        ? await xamanDeposit.mutateAsync({
            token: selectedToken.address,
            amount: rawAmount,
            report: job,
          })
        : await deposit.mutateAsync({
            instructionSender: INSTRUCTION_SENDER,
            token: selectedToken.address,
            amount: rawAmount,
            report: job,
          });
      job.finish({ summary: { tx: formatAddress(tx) } });
      addHistory('DEPOSIT', selectedToken.symbol, amount);
      setAmount('');
    } catch (e) {
      job.fail({ message: e instanceof Error ? e.message : 'Deposit failed' });
    }
  }

  // ---- WITHDRAW ----
  async function handleWithdraw() {
    if (!selectedToken) return;
    if (decimals === undefined) {
      toast('Loading token decimals, please try again', 'error');
      return;
    }
    if (!amount || !address) return;
    let rawAmount: bigint;
    try {
      rawAmount = parseUnits(amount, decimals);
    } catch {
      toast('Invalid amount', 'error');
      return;
    }
    if (rawAmount <= 0n) {
      toast('Amount must be greater than 0', 'error');
      return;
    }
    const job = tray.start({
      title: `Withdraw ${amount} ${selectedToken.symbol}`,
      steps: isXaman ? XAMAN_WITHDRAW_STEPS : WITHDRAW_STEPS,
    });
    try {
      // Withdrawals always pay the connected wallet — a free-typed destination
      // silently kept paying the previously connected wallet across switches.
      const tx = isXaman
        ? await xamanWithdraw.mutateAsync({
            token: selectedToken.address,
            amount: rawAmount,
            to: address,
            report: job,
          })
        : await withdraw.mutateAsync({
            instructionSender: INSTRUCTION_SENDER,
            token: selectedToken.address,
            amount: rawAmount,
            to: address,
            report: job,
          });
      job.finish({ summary: { tx: formatAddress(tx) } });
      addHistory('WITHDRAW', selectedToken.symbol, amount);
      setAmount('');
    } catch (e) {
      job.fail({ message: e instanceof Error ? e.message : 'Withdrawal failed' });
    }
  }

  /** Re-run only the on-chain execute, from the cached/persisted authorization. */
  async function handleRetryExecute() {
    const job = tray.start({
      title: 'Retry withdrawal execution',
      steps: ['Execute on-chain'],
    });
    try {
      const tx = isXaman
        ? await xamanWithdraw.retryExecute(job)
        : await withdraw.retryExecute(INSTRUCTION_SENDER, job);
      job.finish({ summary: { tx: formatAddress(tx) } });
    } catch (e) {
      job.fail({ message: e instanceof Error ? e.message : 'Execute failed' });
    }
  }

  function setMaxDeposit() {
    if (!selectedToken) return;
    const info = tokenInfo[selectedToken.address.toLowerCase()];
    if (info?.balance !== undefined && info.decimals !== undefined) {
      setAmount(parseFloat(formatUnits(info.balance, info.decimals)).toFixed(4));
    }
  }

  function setMaxWithdraw() {
    if (!selectedToken) return;
    const bal = teeBalances[selectedToken.address.toLowerCase()];
    if (bal && decimals !== undefined) {
      try {
        setAmount(
          parseFloat(
            formatUnits(BigInt(Math.floor(bal.available)), decimals)
          ).toFixed(4)
        );
      } catch {
        // ignore
      }
    }
  }

  return (
    <div className="wallet-backdrop" onClick={onClose}>
      <div className="wallet-modal" onClick={e => e.stopPropagation()}>
        {/* Full-width header */}
        <div className="wallet-hdr">
          <div className="wallet-hdr-left">
            <div className="wallet-hdr-eyebrow">FLARE · TESTNET WALLET</div>
            <div className="wallet-hdr-addr">
              {address ? formatAddress(address) : 'NOT CONNECTED'}
            </div>
          </div>
          <div className="wallet-hdr-right">
            {isXaman && <span className="wallet-testnet-badge">XAMAN · FSA</span>}
            <span className="wallet-testnet-badge">TESTNET</span>
            <button className="panel-close" onClick={onClose}>×</button>
          </div>
        </div>

        {/* Left sidebar */}
        <aside className="wallet-side">
          <h3>BALANCES</h3>
          {tokens.map(t => (
            <div key={t.address} className="bal-row">
              <span className="symbol">{t.symbol}</span>
              <span className="amt">{fmtWallet(t.address)}</span>
            </div>
          ))}
          <div style={{ marginTop: 'auto', paddingTop: 12 }}>
            <div
              style={{
                fontSize: 9,
                textTransform: 'uppercase' as const,
                letterSpacing: '0.14em',
                color: 'var(--fg-mute)',
                marginBottom: 4,
              }}
            >
              NETWORK
            </div>
            <div
              style={{
                fontSize: 11,
                display: 'flex',
                alignItems: 'center',
                gap: 6,
              }}
            >
              <span className="dot" />
              <span>COSTON2 TESTNET</span>
            </div>
          </div>
        </aside>

        {/* Right main area */}
        <main className="wallet-main">
          <div className="wallet-tabs">
            {(['FAUCET', 'DEPOSIT', 'WITHDRAW', 'HISTORY'] as Tab[]).map(t => (
              <button
                key={t}
                className={tab === t ? 'active' : ''}
                onClick={() => setTab(t)}
              >
                <span>{t}</span>
                {t === 'FAUCET' && <span className="sub">testnet only</span>}
                {t === 'DEPOSIT' && <span className="sub">on-chain → tee</span>}
                {t === 'WITHDRAW' && <span className="sub">tee → on-chain</span>}
              </button>
            ))}
          </div>

          <div className="wallet-content">
            {tab === 'FAUCET' && (
              <>
                <div className="wallet-note">
                  Mint test tokens directly to your wallet. No real value.
                </div>
                <div
                  style={{ display: 'flex', flexDirection: 'column', gap: 6 }}
                >
                  {tokens.map(t => (
                    <div
                      key={t.address}
                      style={{
                        display: 'flex',
                        justifyContent: 'space-between',
                        fontSize: 11,
                        color: 'var(--fg-dim)',
                      }}
                    >
                      <span style={{ fontWeight: 600, color: 'var(--fg)' }}>
                        {t.symbol}
                      </span>
                      <span>→ {faucetAmountFor(t.symbol)} tokens</span>
                    </div>
                  ))}
                </div>
                <button
                  className="wallet-submit"
                  onClick={handleFaucet}
                  disabled={faucetPending || !address}
                >
                  {faucetPending ? 'MINTING...' : 'MINT ALL TEST TOKENS'}
                </button>
              </>
            )}

            {tab === 'DEPOSIT' && (
              <>
                <div className="wallet-note">
                  Deposit tokens from your wallet into the TEE exchange.
                </div>
                <div className="wallet-seg">
                  {tokens.map((t, i) => (
                    <button
                      key={t.address}
                      className={tokenIdx === i ? 'active' : ''}
                      onClick={() => setTokenIdx(i)}
                    >
                      {t.symbol}
                    </button>
                  ))}
                </div>
                <div
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    fontSize: 12,
                    color: 'var(--fg-dim)',
                  }}
                >
                  <span
                    style={{
                      textTransform: 'uppercase' as const,
                      letterSpacing: '0.12em',
                    }}
                  >
                    WALLET BALANCE
                  </span>
                  <span style={{ color: 'var(--fg)', fontWeight: 500 }}>
                    {selectedToken ? fmtWallet(selectedToken.address) : '—'}{' '}
                    {selectedToken?.symbol}
                  </span>
                </div>
                <div className="wallet-field">
                  <label>AMOUNT ({selectedToken?.symbol})</label>
                  <div style={{ display: 'flex', gap: 6 }}>
                    <input
                      type="number"
                      min="0"
                      step="0.0001"
                      value={amount}
                      onChange={e => setAmount(e.target.value)}
                      placeholder="0.0000"
                      style={{ flex: 1 }}
                    />
                    <button className="hdr-btn" onClick={setMaxDeposit}>
                      MAX
                    </button>
                  </div>
                </div>
                <div className="wallet-preview">
                  <span className="label">DEPOSITING</span>
                  <span className="value">
                    {amount || '0'} {selectedToken?.symbol}
                  </span>
                </div>
                <button
                  className="wallet-submit"
                  onClick={handleDeposit}
                  disabled={depositPending || !amount || !address}
                >
                  {depositPending
                    ? 'PROCESSING...'
                    : `DEPOSIT ${selectedToken?.symbol}`}
                </button>
              </>
            )}

            {tab === 'WITHDRAW' && (
              <>
                <div className="wallet-note">
                  Withdraw tokens from TEE back to your wallet. Requires
                  3-step process.
                </div>
                <div className="wallet-seg">
                  {tokens.map((t, i) => (
                    <button
                      key={t.address}
                      className={tokenIdx === i ? 'active' : ''}
                      onClick={() => setTokenIdx(i)}
                    >
                      {t.symbol}
                    </button>
                  ))}
                </div>
                <div
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    fontSize: 12,
                    color: 'var(--fg-dim)',
                  }}
                >
                  <span
                    style={{
                      textTransform: 'uppercase' as const,
                      letterSpacing: '0.12em',
                    }}
                  >
                    TEE AVAILABLE
                  </span>
                  <span style={{ color: 'var(--fg)', fontWeight: 500 }}>
                    {selectedToken
                      ? fmtTee(selectedToken.address, 'available')
                      : '—'}{' '}
                    {selectedToken?.symbol}
                  </span>
                </div>
                <div className="wallet-field">
                  <label>AMOUNT ({selectedToken?.symbol})</label>
                  <div style={{ display: 'flex', gap: 6 }}>
                    <input
                      type="number"
                      min="0"
                      step="0.0001"
                      value={amount}
                      onChange={e => setAmount(e.target.value)}
                      placeholder="0.0000"
                      style={{ flex: 1 }}
                    />
                    <button className="hdr-btn" onClick={setMaxWithdraw}>
                      MAX
                    </button>
                  </div>
                </div>
                <div className="wallet-field">
                  <label>DESTINATION (CONNECTED WALLET)</label>
                  <input type="text" value={address ?? ''} readOnly disabled />
                </div>

                {(isXaman ? xamanWithdraw.pendingSlip : withdraw.cachedSignature) && !withdrawPending && (
                  <div
                    style={{
                      border: '1px solid var(--line-2)',
                      padding: '10px',
                      background: 'var(--bg-2)',
                    }}
                  >
                    <div
                      style={{
                        fontSize: 10,
                        color: 'var(--fg-mute)',
                        marginBottom: 8,
                        textTransform: 'uppercase' as const,
                        letterSpacing: '0.12em',
                      }}
                    >
                      {isXaman
                        ? 'AUTHORIZATION PENDING — BALANCE ALREADY DEBITED'
                        : 'STEP 3 FAILED — SIGNATURE CACHED'}
                    </div>
                    <button className="wallet-submit" onClick={handleRetryExecute}>
                      RETRY EXECUTE
                    </button>
                  </div>
                )}

                <button
                  className="wallet-submit warn"
                  onClick={handleWithdraw}
                  disabled={withdrawPending || !amount || !address}
                >
                  {withdrawPending
                    ? 'PROCESSING...'
                    : `WITHDRAW ${selectedToken?.symbol}`}
                </button>
              </>
            )}

            {tab === 'HISTORY' && (
              <>
                {history.length === 0 ? (
                  <div className="empty-hint">NO HISTORY YET</div>
                ) : (
                  history.map(h => (
                    <div key={h.id} className="wallet-history-row">
                      <span className="op">
                        {h.type} {h.amount} {h.symbol}
                      </span>
                      <span className="time">
                        {new Date(h.timestamp).toLocaleTimeString()}
                      </span>
                    </div>
                  ))
                )}
              </>
            )}
          </div>
        </main>
      </div>
    </div>
  );
}
