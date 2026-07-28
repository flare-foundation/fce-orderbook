import type { BaseTokenProps } from '../token-icon';

export function HYPE({ size }: BaseTokenProps) {
  return (
    <svg
      width={size || '32'}
      height={size || '32'}
      viewBox="0 0 32 32"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <rect width="32" height="32" rx="16" fill="#072723" />
      <g clip-path="url(#clip0_4495_168025)">
        <path
          d="M25.8907 15.6443C25.8907 22.1842 21.8885 24.2827 19.7797 22.4142C18.044 20.8905 17.5275 17.6709 14.9167 17.3403C11.603 16.9235 11.3161 21.3362 9.1357 21.3362C6.59662 21.3362 6.10889 17.6422 6.10889 15.7449C6.10889 13.8044 6.654 11.1597 8.82009 11.1597C11.3448 11.1597 11.4883 14.94 14.6442 14.7387C17.7858 14.5231 17.8431 10.5848 19.8802 8.90305C21.6589 7.45131 25.8907 9.01803 25.8907 15.6443Z"
          fill="#97FCE4"
        />
      </g>
      <defs>
        <clipPath id="clip0_4495_168025">
          <rect
            width="19.7818"
            height="19.7818"
            fill="white"
            transform="translate(6.10889 5.23633)"
          />
        </clipPath>
      </defs>
    </svg>
  );
}
