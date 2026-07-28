import { Group } from '@mantine/core';
import { TokenIcon, TokenIconName } from '../../atoms/token-icon/token-icon';
import { WalletIcon, WalletIconName } from '../../atoms/wallet-icon/wallet-icon';
import { BodyText } from '../../typography/body-text';
import { tokenType, type TokenType, type WalletType } from '../../types';

export interface ConnectedTokenProps {
  tokens: TokenIconName[] | WalletIconName[];
}

export const ConnectedToken = ({ tokens }: ConnectedTokenProps) => {
  return (
    <Group gap="3n">
      <Group gap="-10px" style={{ flexShrink: 0 }}>
        {tokens.map((item, index) => (
          <div
            style={{
              marginLeft: index !== 0 ? -3 : 0,
              zIndex: tokens.length + 20 - index,
              display: 'flex',
              justifyContent: 'center',
            }}
            key={index}
          >
            {tokenType.includes(item as TokenType) ? (
              <TokenIcon size={20} name={item as TokenType} />
            ) : (
              <WalletIcon size={20} name={item as WalletType} />
            )}
          </div>
        ))}
      </Group>
      <BodyText type="note">Connected</BodyText>
    </Group>
  );
};
