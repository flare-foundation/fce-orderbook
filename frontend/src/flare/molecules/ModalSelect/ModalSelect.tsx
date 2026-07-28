import { Box, Group, Stack } from '@mantine/core';
import { Icon } from '../../atoms/icons';
import { TokenIcon } from '../../atoms/token-icon/token-icon';
import { WalletIcon } from '../../atoms/wallet-icon/wallet-icon';
import { ConnectedToken } from '../../molecules/connected-token/connected-token';
import { ModalPill } from '../../molecules/modal-pill/modal-pill';
import { Link } from '../../typography/link';
import { allTokens, type TokenType, type WalletType } from '../../types';
import classes from './ModalSelect.module.css';

export interface ModalSelectProps {
  status?: 'default' | 'selected' | 'connected' | 'disabled';
  element?: 'network' | 'service';
  wallet?: WalletType;
  tokens?: TokenType[];
  withQrCode?: boolean;
  onSelect?: () => void;
  onDisconnect?: () => void;
}
export const ModalSelect = ({
  status,
  element,
  wallet,
  tokens,
  withQrCode,
  onSelect,
  onDisconnect,
}: ModalSelectProps) => {
  return (
    <Box
      className={classes.box}
      data-status={status}
      data-element={element}
      py={{ base: 'xs', sm: 'lg' }}
      px={{ base: 'xs', sm: 'xl' }}
      onClick={status !== 'disabled' && status !== 'connected' ? onSelect : undefined}
      style={{ cursor: status !== 'disabled' && status !== 'connected' ? 'pointer' : undefined }}
    >
      <Group justify="space-between" pr="32px">
        <Group gap="md">
          {element === 'service' && wallet ? (
            <>
              <Box visibleFrom="sm">
                {' '}
                <WalletIcon size={60} name={wallet} />
              </Box>
              <Box hiddenFrom="sm">
                {' '}
                <WalletIcon size={48} name={wallet} />
              </Box>
            </>
          ) : tokens ? (
            <>
              <Box
                visibleFrom="sm"
                style={{ height: '80px', opacity: status === 'connected' ? 0.5 : 1 }}
              >
                <TokenIcon size={80} name={tokens[0]} />
              </Box>
              <Box
                hiddenFrom="sm"
                style={{ height: '48px', opacity: status === 'connected' ? 0.5 : 1 }}
              >
                <TokenIcon size={48} name={tokens[0]} />
              </Box>
            </>
          ) : (
            <></>
          )}

          <Stack gap="n">
            <Link
              fz={{ base: 'bodyLink', sm: 'lLink' }}
              lh="largeLink"
              data-status={status}
            >
              {element === 'service' ? wallet : tokens && allTokens[tokens[0]].longName}
            </Link>
            {status === 'connected' && element === 'service' && tokens && (
              <ConnectedToken tokens={tokens} />
            )}
            {status === 'connected' && element === 'network' && wallet && (
              <ConnectedToken tokens={[wallet]} />
            )}
          </Stack>
        </Group>
        {element === 'service' && withQrCode && status !== 'connected' && (
          <ModalPill label="QRCode" />
        )}
        {status === 'connected' && (
          <Link type="smallBodyLink" leftIcon={Icon.Disconnect} onClick={onDisconnect}>
            Disconnect
          </Link>
        )}
      </Group>
      {status === 'selected' && (
        <>
          <Box className={classes.iconWrapper}>
            <Icon.Check color="var(--mantine-color-neutrals-fills-white)" size={14} />
          </Box>
        </>
      )}
    </Box>
  );
};
