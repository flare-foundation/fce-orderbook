import type { ReactNode } from 'react';

export interface IconProps {
  variant?: 'default' | 'gray' | 'pink';
  size?: number | string;
  className?: string;
  color?: string;
}

const colorVariant = {
  default: 'var(--mantine-color-neutrals-fills-dark)',
  gray: 'var(--mantine-color-neutrals-fills-medium)',
  pink: 'var(--mantine-color-pinks-fills-normal-brand)',
};

interface BaseIconProps extends IconProps {
  children: (fill: string) => ReactNode;
  viewBox?: string;
}

export const BaseIcon = ({
  size = '20',
  color,
  variant = 'default',
  className,
  children,
  viewBox = '0 0 20 20',
}: BaseIconProps) => (
  <svg
    width={size}
    height={size}
    viewBox={viewBox}
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
    className={className}
  >
    {children(color || colorVariant[variant])}
  </svg>
);
