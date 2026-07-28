import type { WalletIconProps } from '../wallet-icon';

export function LedgerInv({ size = 44 }: WalletIconProps) {
  return (
    <svg
      width={size}
      preserveAspectRatio="xMidYMid meet"
      viewBox="0 0 44 44"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <rect width="44" height="44" rx={22} fill="#101010" />
      <path
        d="M8 28.0495V36H18.532V34.2369H9.53454V28.0495H8ZM34.4655 28.0495V34.2369H25.468V35.9996H36V28.0495H34.4655ZM18.5473 15.9505V28.0491H25.468V26.459H20.0818V15.9505H18.5473ZM8 8V15.9505H9.53454V9.76278H18.532V8H8ZM25.468 8V9.76278H34.4655V15.9505H36V8H25.468Z"
        fill="#D1D1D1"
      />
    </svg>
  );
}
