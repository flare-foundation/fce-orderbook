import { Grid, Stack } from '@mantine/core';
import { Icon } from '../../atoms/icons';
import { TokenIcon } from '../../atoms/token-icon/token-icon';
import { WalletIcon } from '../../atoms/wallet-icon/wallet-icon';
import type { TokenType, WalletType } from '../../types';
import classes from './GridConnect.module.css';

export interface GridConnectWalletProps {
  walletsSupportedNetworks: { wallet: WalletType; supportedNetworks: TokenType[] }[];
  networks: TokenType[];
}

export const GridConnectWallet = ({
  walletsSupportedNetworks,
  networks,
}: GridConnectWalletProps) => {
  return (
    <Stack gap="0">
      <Grid
        columns={networks.length + 1}
        w={(networks.length + 1) * 44}
        gutter={0}
        classNames={{ col: classes.col }}
      >
        <Grid.Col span={1}> </Grid.Col>
        {networks.map((item) => (
          <Grid.Col span={1} key={item}>
            <TokenIcon name={item} size={24} />
          </Grid.Col>
        ))}
      </Grid>

      {walletsSupportedNetworks.map((item, walletIndex) => (
        <Grid
          w={(networks.length + 1) * 44}
          columns={networks.length + 1}
          key={walletIndex}
          gutter={0}
          classNames={{ col: classes.col }}
        >
          <Grid.Col span={1}>
            <WalletIcon name={item.wallet} size={24} />
          </Grid.Col>
          {networks.map((network, index) => (
            <Grid.Col
              key={index}
              span={1}
              style={{
                ...(walletIndex !== 0 && {
                  borderTop: '1px solid var(--mantine-color-token-pairs-neutral-light)',
                }),
                ...(index !== 0 && {
                  borderLeft: '1px solid var(--mantine-color-token-pairs-neutral-light)',
                }),
                alignContent: 'center',
                flexWrap: 'wrap',
              }}
            >
              {item.supportedNetworks?.includes(network) ? <Icon.Check size={10} /> : <></>}
            </Grid.Col>
          ))}{' '}
        </Grid>
      ))}
    </Stack>
  );
};
