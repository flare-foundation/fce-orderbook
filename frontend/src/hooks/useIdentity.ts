import { useAccount } from 'wagmi';
import type { Address } from 'viem';
import { useXaman } from '../contexts/XamanContext';

export type WalletKind = 'metamask' | 'xaman' | null;

/**
 * The single source of "who is the user" for both wallet models:
 *  - metamask (any EVM wallet via RainbowKit): the connected EOA.
 *  - xaman: the PersonalAccount derived from the connected XRPL address.
 *
 * Everything that previously read useAccount().address should read this, so
 * orders, TEE state, and balances all key off the same identity.
 */
export function useIdentity(): {
  address: Address | undefined;
  walletKind: WalletKind;
  isConnected: boolean;
} {
  const xaman = useXaman();
  const { address, isConnected } = useAccount();

  if (xaman.pa) {
    return { address: xaman.pa, walletKind: 'xaman', isConnected: true };
  }
  return {
    address,
    walletKind: isConnected && address ? 'metamask' : null,
    isConnected: isConnected && !!address,
  };
}
