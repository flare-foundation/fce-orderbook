import { Group, Stack } from '@mantine/core';
import { TokenIcon, TokenIconName } from '../../atoms/token-icon/token-icon';
import { BodyText } from '../../typography/body-text';

export interface TokenBalanceProps {
  token: TokenIconName;
  /** Formatted amount string (e.g., "1,234.56") */
  amountFormatted: string;
  /** Formatted fiat value string (e.g., "$1,234.56") */
  fiatFormatted: string;
}

export const TokenBalance = ({ token, fiatFormatted, amountFormatted }: TokenBalanceProps) => {
  return (
    <Group gap="sm" wrap="nowrap">
      <TokenIcon name={token} size={56} />
      <Stack gap="0" w="100%">
        <BodyText c="var(--mantine-color-neutrals-fills-dark)" type="lBody">
          {token}
        </BodyText>
        <Group justify="space-between">
          <BodyText c="var(--mantine-color-neutrals-fills-medium)">{amountFormatted}</BodyText>
          <BodyText c="var(--mantine-color-neutrals-fills-medium)">{fiatFormatted}</BodyText>
        </Group>
      </Stack>
    </Group>
  );
};
