import * as TokenIcons from './icons';

export type TokenIconName = keyof typeof TokenIcons;

export interface BaseTokenProps {
  size?: number;
  radius?: number | string;
  disabled?: boolean;
  withBorder?: boolean;
}

export interface FAssetTokenProps extends BaseTokenProps {
  withIcon?: boolean;
}

export interface TokenIconProps extends BaseTokenProps {
  name: TokenIconName;
  withIcon?: boolean;
}

const TokenIconComponent = ({ name, withIcon, ...rest }: TokenIconProps) => {
  const Icon = TokenIcons[name];
  const props = { ...rest, withIcon };
  return <Icon {...props} />;
};

export const TokenIcon = Object.assign(TokenIconComponent, {
  ...TokenIcons,
});

export default TokenIcon;
