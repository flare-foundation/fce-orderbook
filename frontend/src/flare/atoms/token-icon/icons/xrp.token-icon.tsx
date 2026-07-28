import type { BaseTokenProps } from '../token-icon';

export function XRP({ size, radius, withBorder = false }: BaseTokenProps) {
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
        fill="#222222"
      />
      {withBorder && (
        <rect
          x="0.254883"
          y="0.249512"
          width="31.5"
          height="31.5"
          rx={radius || '15.75'}
          stroke="white"
          strokeWidth="0.5"
        />
      )}
      <path
        d="M12.0596 18.3589C14.2389 16.2004 17.7709 16.2004 19.9502 18.3589L26.0049 24.3511H23.1152L18.501 19.7866C17.1217 18.4212 14.895 18.4212 13.5088 19.7866L8.89453 24.3511H6.00488L12.0596 18.3589ZM13.5127 12.1724C14.8919 13.5376 17.1197 13.5375 18.5059 12.1724L23.0781 7.64795H25.9678L19.9541 13.606C17.7748 15.7576 14.2438 15.7576 12.0645 13.606L6.04395 7.64795H8.94043L13.5127 12.1724Z"
        fill="white"
      />
    </svg>
  );
}
