import { Group, Stack } from '@mantine/core';
import { TokenIcon } from '../../atoms/token-icon/token-icon';
import { BodyText } from '../../typography/body-text';
import type { FassetType, TokenType } from '../../types';

export interface IconWithValueProps {
  value: string | number;
  token: TokenType | FassetType;
  balance: string | number;
}

export const IconWithValue = ({ value, token, balance }: IconWithValueProps) => (
  <Stack gap="0" py="2n">
    <Group gap="3n">
      <TokenIcon name={token} size={18} />

      <BodyText c="var(--mantine-color-neutrals-fills-dark)">{balance}</BodyText>
    </Group>
    <BodyText type="sBody" c="var(--mantine-color-neutrals-fills-medium)">
      ${value}
    </BodyText>
  </Stack>
);
