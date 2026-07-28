import { Box, Group, Stack } from '@mantine/core';
import { Button } from '../../atoms/button/button';
import { WalletIcon } from '../../atoms/wallet-icon/wallet-icon';
import { ModalHeader } from '../../molecules/ModalHeader/ModalHeader';
import { BodyText } from '../../typography/body-text';
import classes from './ModalXamanSign.module.css';

// App-specific Xaman QR signing step (connect SignIn, account activation)
// modeled on the design system's ModalConnectToNetwork organism: same chrome
// and layout, with the centered TokenIcon swapped for the Xaman QR code (the
// DS has no QR component).
export interface ModalXamanSignProps {
  onClose: () => void;
  qrPng?: string | null;
  deeplink?: string | null;
  headerLabel?: string;
  primaryText?: string;
  secondaryText?: string;
}
export const ModalXamanSign = ({
  onClose,
  qrPng,
  deeplink,
  headerLabel = 'Connect Wallet',
  primaryText = 'Scan with the Xaman app',
  secondaryText = 'Open Xaman on your phone and approve the SignIn request',
}: ModalXamanSignProps) => {
  return (
    <Box className={classes.wrapper}>
      <Stack px="xl" py="lg" gap="xl">
        <ModalHeader label={headerLabel} onClose={onClose} />
        <Group gap="sm">
          <WalletIcon size={42} name="Xaman" />
          <BodyText type="sBody" c="var(--mantine-color-neutrals-fills-medium)">
            Xaman
          </BodyText>
        </Group>
      </Stack>
      <Stack gap="md" px="xl" align="center">
        <Box pb="xl2">
          {qrPng ? (
            <img
              src={qrPng}
              width={180}
              height={180}
              alt="Xaman QR code"
              style={{ display: 'block', borderRadius: 8 }}
            />
          ) : (
            <Box
              w={180}
              h={180}
              style={{ display: 'flex', alignItems: 'center', justifyContent: 'center' }}
            >
              <BodyText type="sBody" c="var(--mantine-color-neutrals-fills-medium)">
                Preparing request…
              </BodyText>
            </Box>
          )}
        </Box>
        <Stack gap="2n" align="center" pb="xl">
          <BodyText type="lBody">{primaryText}</BodyText>
          <BodyText c="var(--mantine-color-neutrals-fills-medium)">{secondaryText}</BodyText>
        </Stack>
      </Stack>
      {deeplink && (
        <Stack align="center" px="xl" pb="xl2">
          <Button variant="secondary" component="a" href={deeplink} target="_blank" rel="noreferrer">
            Open in Xaman
          </Button>
        </Stack>
      )}
    </Box>
  );
};
