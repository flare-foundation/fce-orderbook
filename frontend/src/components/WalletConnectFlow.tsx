// WalletConnectFlow — the Flare Design System wallet picker + Xaman connect
// flow, ported from shielded-transfer's shell.jsx. One CONNECT button in the
// header opens: pick a wallet (ModalConnectWallet) → for Xaman, scan the
// SignIn/bind QRs (ModalXamanSign) → success (ModalSuccess). MetaMask connects
// via wagmi's connector directly — no RainbowKit modal, and no silent
// auto-reconnect on page load.

import { useEffect, useState } from 'react';
import { Modal } from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { useAccount, useConnect, useDisconnect } from 'wagmi';
import { useXaman } from '../contexts/XamanContext';
import { useToast } from './ui/Toast';
import { ModalConnectWallet } from '../flare/organisms/ModalConnectWallet/ModalConnectWallet';
import { ModalXamanSign } from '../flare/organisms/ModalXamanSign/ModalXamanSign';
import { ModalSuccess } from '../flare/organisms/ModalSuccess/ModalSuccess';
import { WalletIcon } from '../flare/atoms/wallet-icon/wallet-icon';
import type { WalletType } from '../flare/types';

function truncate(addr: string): string {
  return addr.length > 12 ? `${addr.slice(0, 6)}…${addr.slice(-4)}` : addr;
}

export function WalletConnectButton() {
  const xaman = useXaman();
  const { address: evmAddress, isConnected: evmConnected } = useAccount();
  const { connectAsync, connectors } = useConnect();
  const { disconnect: disconnectEvm } = useDisconnect();
  const { toast } = useToast();

  const [opened, { open, close }] = useDisclosure(false);
  const [flow, setFlow] = useState<'select' | 'xaman' | 'success'>('select');

  const openConnect = () => {
    setFlow('select');
    open();
  };

  async function select(wallet: WalletType) {
    // Clicking the already-connected wallet disconnects it.
    if (wallet === 'Xaman' && xaman.xrplAddress) {
      xaman.disconnect();
      close();
      return;
    }
    if (wallet === 'MetaMask' && evmConnected) {
      disconnectEvm();
      close();
      return;
    }

    // Switching: drop the currently connected wallet before connecting the new one.
    if (xaman.xrplAddress) xaman.disconnect();
    if (evmConnected) disconnectEvm();

    if (wallet === 'MetaMask') {
      close();
      const metamask =
        connectors.find(c => c.name === 'MetaMask') ??
        connectors.find(c => c.type === 'injected') ??
        connectors[0];
      try {
        await connectAsync({ connector: metamask });
        toast('MetaMask connected', 'success');
      } catch (e) {
        toast(e instanceof Error ? e.message.split('\n')[0] : 'Connect failed', 'error');
      }
    } else if (wallet === 'Xaman') {
      setFlow('xaman');
      try {
        await xaman.connect();
        setFlow('success');
      } catch (e) {
        close();
        toast(e instanceof Error ? e.message : 'Xaman connect failed', 'error');
      }
    }
  }

  // The flow modal owns QR rendering while it's open; keep it in sync when
  // the connect() promise moves from the SignIn scan to the bind scan.
  useEffect(() => {
    if (!opened && flow !== 'select') setFlow('select');
  }, [opened, flow]);

  // Connected chip: click reopens the picker to switch wallets (the connected
  // one is marked and clicking it disconnects). Xaman wins the identity when
  // both are connected, mirroring useIdentity.
  const chipLabel = xaman.xrplAddress
    ? truncate(xaman.xrplAddress)
    : evmConnected && evmAddress
      ? truncate(evmAddress)
      : null;

  return (
    <>
      <button
        className="hdr-user"
        onClick={openConnect}
        disabled={xaman.connecting}
        title={chipLabel ? 'Switch or disconnect wallet' : undefined}
      >
        {xaman.connecting ? 'CONNECTING…' : chipLabel ?? 'CONNECT'}
      </button>
      <Modal
        opened={opened}
        onClose={close}
        centered
        size="auto"
        padding={0}
        withCloseButton={false}
        overlayProps={{ backgroundOpacity: 0.6, blur: 2 }}
        styles={{
          content: { background: 'transparent', boxShadow: 'none' },
          body: { padding: 0 },
        }}
      >
        {flow === 'select' ? (
          <ModalConnectWallet
            onClose={close}
            wallets={[
              { wallet: 'MetaMask', status: evmConnected ? 'connected' : 'default' },
              { wallet: 'Xaman', withQrCode: true, status: xaman.xrplAddress ? 'connected' : 'default' },
            ]}
            onWalletSelect={select}
          />
        ) : flow === 'xaman' ? (
          <ModalXamanSign
            onClose={close}
            qrPng={xaman.xamanRef?.qrPng}
            deeplink={xaman.xamanRef?.deeplink}
          />
        ) : (
          <ModalSuccess
            onClose={close}
            headerText="Connect Wallet"
            LeftIcon={WalletIcon.Xaman}
            leftText="Xaman"
            ContentIcon={WalletIcon.Xaman}
            contentPrimaryText="Wallet connected"
            contentSecondaryText={`Signed in as ${xaman.xrplAddress ?? ''}`}
            buttonText="Done"
            onClick={close}
          />
        )}
      </Modal>
    </>
  );
}
