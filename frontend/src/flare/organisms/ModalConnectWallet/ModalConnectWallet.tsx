import { Box, Flex, Stack } from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { GridConnectWallet } from '../../molecules/GridConnectWallet/GridConnectWallet';
import { ModalHeader } from '../../molecules/ModalHeader/ModalHeader';
import { ModalNote } from '../../molecules/ModalNote/ModalNote';
import { ModalSelect } from '../../molecules/ModalSelect/ModalSelect';
import { BodyText } from '../../typography/body-text';
import type { TokenType, WalletType } from '../../types';
import classes from './ModalConnectWallet.module.css';

export interface ModalConnectWallet {
  onClose: () => void;
  wallets: {
    wallet: WalletType;
    supportedNetworks?: TokenType[];
    selectedNetworks?: TokenType[];
    withQrCode?: boolean;
    status?: 'default' | 'selected' | 'connected' | 'disabled';
  }[];
  onWalletSelect: (wallet: WalletType) => void;
  networks?: TokenType[];
}
export const ModalConnectWallet = ({
  onClose,
  wallets,
  onWalletSelect,
  networks,
}: ModalConnectWallet) => {
  const [modalNoteOpened, { toggle: toggleModalNote }] = useDisclosure(false);
  return (
    <Box className={classes.wrapper}>
      <Stack gap="xl" px="xl" py={{ base: 'xl2', sm: 'lg' }}>
        {' '}
        <ModalHeader label="Connect Wallet" onClose={onClose} />
        <BodyText c="var(--mantine-color-neutrals-fills-medium)">
          Select a wallet to connect.
        </BodyText>
      </Stack>
      <Stack gap="0">
        {wallets.map((wallet, index) => (
          <ModalSelect
            {...wallet}
            element="service"
            key={index}
            onSelect={() => onWalletSelect(wallet.wallet)}
          />
        ))}
      </Stack>

      {networks && (
        <Box
          px="xl"
          py="lg"
          onClick={toggleModalNote}
          style={{ cursor: 'pointer', display: 'inline-block' }}
        >
          <ModalNote opened={modalNoteOpened} label="Which wallet should I use?">
            <Flex gap="28px" direction={{ base: 'column', sm: 'row' }} align="center">
              <GridConnectWallet
                walletsSupportedNetworks={wallets.map((wallet) => ({
                  wallet: wallet.wallet,
                  supportedNetworks: wallet.supportedNetworks || [],
                }))}
                networks={networks}
              />
              <BodyText type="sBody" c="var(--mantine-color-neutrals-fills-medium)">
                *Note - You can only connect to each network once. To connect a new wallet to a
                network you are already connected to, you must first disconnect from that network.
              </BodyText>
            </Flex>
          </ModalNote>
        </Box>
      )}
    </Box>
  );
};
