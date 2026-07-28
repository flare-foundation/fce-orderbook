import * as WalletIcons from './icons';

export type WalletIconName = keyof typeof WalletIcons;

export interface WalletIconProps {
  size?: number;
  radius?: number | string;
  disabled?: boolean;
}

export interface WalletIconWithNameProps extends WalletIconProps {
  name: WalletIconName;
}

const WalletIconComponent = ({ name, ...rest }: WalletIconWithNameProps) => {
  if (!name) {
    return null;
  }
  const Icon = WalletIcons[name];
  return <Icon {...rest} />;
};

export const WalletIcon = Object.assign(WalletIconComponent, {
  ...WalletIcons,
});

export default WalletIcon;
