import type { BaseTokenProps } from '../token-icon';

export function Doge({ size, radius }: BaseTokenProps) {
  return (
    <svg
      width={size || '32'}
      height={size || '32'}
      viewBox="0 0 32 32"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      style={{ flexShrink: 0 }}
    >
      <rect x="0.25" y="0.25" width="31.5" height="31.5" rx={radius || '15.75'} fill="#C2A633" />
      <rect
        x="0.25"
        y="0.25"
        width="31.5"
        height="31.5"
        rx={radius || '15.75'}
        stroke="white"
        strokeWidth="0.5"
      />
      <path
        d="M15.0977 8.00011C16.1865 8.00005 23.3866 7.7752 23.3867 16.128C23.3867 24.6076 15.8789 23.9912 15.8564 23.9894H10.501V16.8673H8.61328V15.1222H10.501V8.00011H15.0977ZM13.5156 15.1232H16.8398V16.8673H13.5156V21.0343H15.7324C16.3028 21.0343 20.4075 21.0978 20.4014 16.1866C20.395 11.2749 16.4197 10.9562 15.6289 10.9562H13.5156V15.1232Z"
        fill="white"
      />
    </svg>
  );
}
