import type { BaseTokenProps } from '../token-icon';

export function USDT0({ size, radius }: BaseTokenProps) {
  return (
    <svg
      width={size || '32'}
      height={size || '32'}
      viewBox="0 0 32 32"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      style={{ flexShrink: 0 }}
    >
      <rect
        x="0.254883"
        y="0.249512"
        width="31.5"
        height="31.5"
        rx={radius || '15.75'}
        fill="#00B988"
      />
      <rect
        x="0.254883"
        y="0.249512"
        width="31.5"
        height="31.5"
        rx={radius || '15.75'}
        stroke="white"
        strokeWidth="0.5"
      />
      <path
        d="M7.97803 6.99951H24.0321V10.6481H17.7889V14.2968H14.3024V21.4319H17.9513L17.9507 14.2968H21.5996V21.4319H17.9513V24.9995H14.3024V21.5941H10.8969V14.2968H14.2213V10.6481H7.97803V6.99951Z"
        fill="white"
      />
    </svg>
  );
}
