import {
  Anchor,
  createPolymorphicComponent,
  Group,
  PolymorphicComponentProps,
  type AnchorProps,
} from '@mantine/core';
import { Icon, type IconProps } from '../atoms/icons';

type LinkType = 'lLink' | 'bodyLink' | 'smallBodyLink' | 'noteTextLink';
type VariantType = 'normal' | 'dark';
export type LinkOwnProps = Omit<React.ComponentPropsWithoutRef<'a'>, keyof AnchorProps> &
  AnchorProps & {
    children: React.ReactNode;
    type?: 'lLink' | 'bodyLink' | 'smallBodyLink' | 'noteTextLink';
    onClick?: () => void;
    leftIcon?: React.ElementType<IconProps>;
    rightIcon?: React.ElementType<IconProps>;
    href?: string;
    color?: string;
    isExternal?: boolean;
    variant?: 'dark' | 'normal';
  };

export type LinkProps<C extends React.ElementType = 'a'> = PolymorphicComponentProps<
  C,
  LinkOwnProps
>;

const ICON_SIZES: Record<LinkType, number> = {
  lLink: 26,
  bodyLink: 20,
  smallBodyLink: 10,
  noteTextLink: 16,
};

const VARIANT_STYLES: Record<
  VariantType,
  { color: string; fontWeight: number; letterSpacing: string | number }
> = {
  dark: {
    color: 'var(--mantine-color-pinks-fills-darker)',
    fontWeight: 700,
    letterSpacing: '0.03em',
  },
  normal: {
    color: 'var(--mantine-color-pinks-fills-normal-brand)',
    fontWeight: 500,
    letterSpacing: 0,
  },
};

const _Link = ({
  ref,
  type = 'bodyLink',
  children,
  color,
  leftIcon: LeftIcon,
  rightIcon: RightIcon,
  isExternal,
  variant = 'dark',

  ...rest
}: LinkProps) => {
  const iconSize = ICON_SIZES[type];
  const _color = color || VARIANT_STYLES[variant].color;
  const fw = VARIANT_STYLES[variant].fontWeight;
  const lts = VARIANT_STYLES[variant].letterSpacing;
  return (
    <Anchor
      ref={ref}
      fz={type}
      lh={type === 'lLink' ? 'largeLink' : type}
      lts={lts}
      c={_color}
      fw={fw}
      td="none"
      {...rest}
    >
      <Group gap="3n" wrap="nowrap">
        {LeftIcon && <LeftIcon size={iconSize} color={_color} />}
        {children}
        {isExternal ? (
          <Icon.LinkExternal size={iconSize} color={_color} />
        ) : (
          RightIcon && <RightIcon size={iconSize} color={_color} />
        )}
      </Group>
    </Anchor>
  );
};

export const Link = createPolymorphicComponent<'a', LinkOwnProps>(_Link);
