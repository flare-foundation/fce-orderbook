import type { BaseTokenProps } from '../token-icon';

export function ETH({ size, radius }: BaseTokenProps) {
  return (
    <svg
      width={size || 32}
      height={size || 32}
      viewBox="0 0 32 32"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      style={{ flexShrink: 0 }}
    >
      <rect
        x="0.254883"
        y="0.25"
        width="31.5"
        height="31.5"
        rx={radius || '15.75'}
        fill="#ECEFF0"
      />
      <rect
        x="0.254883"
        y="0.25"
        width="31.5"
        height="31.5"
        rx={radius || '15.75'}
        stroke="white"
        strokeWidth="0.5"
      />
      <path d="M16.0029 6V13.3932L22.2518 16.1855L16.0029 6Z" fill="#343434" />
      <path d="M16.0031 6L9.75342 16.1855L16.0031 13.3932V6Z" fill="#8C8C8C" />
      <path d="M16.0029 20.9766V26.0001L22.2559 17.3491L16.0029 20.9766Z" fill="#3C3C3B" />
      <path d="M16.0031 26.0001V20.9757L9.75342 17.3491L16.0031 26.0001Z" fill="#8C8C8C" />
      <path d="M16.0029 19.8134L22.2518 16.1851L16.0029 13.3945V19.8134Z" fill="#141414" />
      <path d="M9.75342 16.1851L16.0031 19.8134V13.3945L9.75342 16.1851Z" fill="#393939" />
    </svg>
  );
}
