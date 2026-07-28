import { type ReactNode } from 'react';
import { Group } from '@mantine/core';
import { CopyButton } from '../../atoms/copy-button/copy-button';
import { Label } from '../../typography/label';
import { truncateString } from '../../utils';

export interface ConnectedWalletWithAddressProps {
  walletIcon: ReactNode;
  address: string;
}
export const ConnectedWalletWithAddress = ({
  walletIcon,
  address,
}: ConnectedWalletWithAddressProps) => {
  return (
    <Group gap="xs">
      <Group gap="xs">
        {walletIcon}
        <Label c="var(--mantine-color-neutrals-fills-medium)" type="lLabel">
          {truncateString(address, 4, 4)}{' '}
        </Label>
        <CopyButton value={address} />
      </Group>
    </Group>
  );
};
