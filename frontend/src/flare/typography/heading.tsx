import { type ReactNode } from 'react';
import { Title as MantineTitle, type TitleOrder, type TitleProps } from '@mantine/core';

interface HeaderProps extends Omit<TitleProps, 'order'> {
  order: TitleOrder | 7;
  children: ReactNode;
  c?: string;
}

export function Heading({ order, children, c, ...rest }: HeaderProps) {
  return (
    <MantineTitle
      c={c || 'var(--mantine-color-token-pairs-neutral-dark)'}
      order={order === 7 ? 6 : order}
      lh={`h${order}`}
      fz={`var(--mantine-h${order}-font-size)`}
      {...rest}
    >
      {children}
    </MantineTitle>
  );
}
