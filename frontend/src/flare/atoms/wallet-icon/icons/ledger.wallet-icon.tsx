import type { WalletIconProps } from '../wallet-icon';

export function Ledger({ size = 32 }: WalletIconProps) {
  return (
    <svg
      preserveAspectRatio="xMidYMid meet"
      width={size}
      viewBox="0 0 32 32"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M2 22.0495V30H12.532V28.2369H3.53454V22.0495H2ZM28.4655 22.0495V28.2369H19.468V29.9996H30V22.0495H28.4655ZM12.5473 9.95051V22.0491H19.468V20.459H14.0818V9.95051H12.5473ZM2 2V9.95051H3.53454V3.76278H12.532V2H2ZM19.468 2V3.76278H28.4655V9.95051H30V2H19.468Z"
        fill="var(--mantine-color-neutrals-fills-dark)"
      />
    </svg>
  );
}
