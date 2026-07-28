export const tokenType = [
  'BTC',
  'Doge',
  'ETH',
  'SGB',
  'USDC',
  'USDT',
  'USDX',
  'XRP',
  'FLR',
  'USDT0',
  'CFLR',
  'C2FLR',
] as const;

export type TokenType = (typeof tokenType)[number];

interface TokenProps {
  shortName: string;
  longName: string;
}

export const allTokens: Record<TokenType, TokenProps> = {
  BTC: {
    shortName: 'BTC',
    longName: 'Bitcoin',
  },
  Doge: {
    shortName: 'Doge',
    longName: 'Doge',
  },
  ETH: {
    shortName: 'ETH',
    longName: 'Ethereum',
  },
  SGB: {
    shortName: 'SGB',
    longName: 'Songbird',
  },
  USDC: {
    shortName: 'USDC',
    longName: 'USD Coin',
  },
  USDT: {
    shortName: 'USDT',
    longName: 'Tether',
  },
  USDX: {
    shortName: 'USDX',
    longName: 'USDX Stablecoin',
  },
  XRP: {
    shortName: 'XRP',
    longName: 'Ripple',
  },
  FLR: {
    shortName: 'FLR',
    longName: 'Flare',
  },
  USDT0: {
    shortName: 'USDT0',
    longName: 'Tether',
  },
  CFLR: {
    shortName: 'CFLR',
    longName: 'Coston',
  },
  C2FLR: {
    shortName: 'C2FLR',
    longName: 'Coston 2',
  },
};

export const fassetType = ['FBTC', 'FDoge', 'FXRP', 'StXRP'] as const;
export type FassetType = (typeof fassetType)[number];

export const walletType = [
  'WalletConnect',
  'MetaMask',
  'Trezor',
  'Ledger',
  'Bifrost',
  'Xaman',
  'Dcent',
] as const;
export type WalletType = (typeof walletType)[number];

export const serviceType = [
  'Firelight',
  'Kinetic',
  'Blazeswap',
  'Enosys',
  'SparkDex',
  'StratEx',
] as const;

export type ServiceType = (typeof serviceType)[number];

export const vaultType = [
  'firelight',
  'trezor',
  'xaman',
  'walletConnect',
  'ledger',
  'upshift',
] as const;
