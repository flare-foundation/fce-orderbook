import type { BaseTokenProps } from '../token-icon';

export function ETHInv({ size, radius }: BaseTokenProps) {
  return (
    <svg
      width={size || 32}
      height={size || 32}
      viewBox="0 0 32 32"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      style={{ flexShrink: 0 }}
    >
      <rect x="0.00463867" width="32" height="32" rx={radius || 22} fill="#627EEA" />
      <path
        d="M16 32C24.8366 32 32 24.8366 32 16C32 7.16344 24.8366 0 16 0C7.16344 0 0 7.16344 0 16C0 24.8366 7.16344 32 16 32Z"
        fill="#627EEA"
      />
      <path
        d="M16.498 3.99991V12.8699L23.995 16.2199L16.498 3.99991Z"
        fill="white"
        fill-opacity="0.602"
      />
      <path d="M16.498 3.99995L9 16.22L16.498 12.87V3.99995Z" fill="white" />
      <path
        d="M16.498 21.9678V27.9948L24 17.6158L16.498 21.9678Z"
        fill="white"
        fill-opacity="0.602"
      />
      <path d="M16.498 27.9948V21.9668L9 17.6158L16.498 27.9948Z" fill="white" />
      <path
        d="M16.498 20.5728L23.995 16.2198L16.498 12.8718V20.5728Z"
        fill="white"
        fill-opacity="0.2"
      />
      <path d="M9 16.2198L16.498 20.5728V12.8718L9 16.2198Z" fill="white" fill-opacity="0.602" />
    </svg>
  );
}
