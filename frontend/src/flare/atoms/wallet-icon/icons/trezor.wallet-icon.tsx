import type { WalletIconProps } from '../wallet-icon';

export function Trezor({ size = 28 }: WalletIconProps) {
  return (
    <svg
      width={size}
      preserveAspectRatio="xMidYMid meet"
      viewBox="0 0 28 28"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M20.8541 6.49814C20.8541 2.94643 17.787 0 14.0535 0C10.32 0 7.25289 2.94791 7.25289 6.49814V8.57514H4.45459V23.5145L14.0535 28L23.6546 23.5115V8.63878H20.8563L20.8541 6.49814ZM10.7198 6.49814C10.7198 4.82366 12.1867 3.48363 14.0535 3.48363C15.9202 3.48363 17.3871 4.82366 17.3871 6.49814V8.57514H10.7198V6.49814ZM19.7871 21.1023L14.0535 23.7824L8.31992 21.1023V12.1254H19.7871V21.1023Z"
        fill="var(--mantine-color-neutrals-fills-dark)"
      />
    </svg>
  );
}
