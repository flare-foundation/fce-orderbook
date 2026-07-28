import { PolymorphicComponentProps, Text, TextProps } from '@mantine/core';

export type LabelProps<C extends React.ElementType = 'p'> = PolymorphicComponentProps<
  C,
  TextProps
> & {
  type?: 'xlLabel' | 'lLabel' | 'label' | 'sLabel';
};

export function Label({ type = 'label', c, children, ...rest }: LabelProps) {
  return (
    <Text
      fz={type}
      lh={type}
      fw={700}
      lts="0.1em"
      c={c || 'var(--mantine-color-token-pairs-neutral-dark)'}
      {...rest}
    >
      {children}
    </Text>
  );
}
