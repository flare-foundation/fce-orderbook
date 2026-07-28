import { PolymorphicComponentProps, Text, TextProps } from '@mantine/core';

export type BodyTextProps<C extends React.ElementType = 'p'> = PolymorphicComponentProps<
  C,
  TextProps
> & {
  type?: 'lBody' | 'sBody' | 'body' | 'note';
};

export function BodyText({ type = 'body', c, children, ...rest }: BodyTextProps) {
  return (
    <Text
      fz={type}
      lh={type}
      fw={500}
      c={c || 'var(--mantine-color-token-pairs-neutral-dark)'}
      {...rest}
    >
      {children}
    </Text>
  );
}
