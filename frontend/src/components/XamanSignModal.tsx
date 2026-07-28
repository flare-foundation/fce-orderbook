import { Modal } from '@mantine/core';
import { useXaman } from '../contexts/XamanContext';
import { ModalXamanSign } from '../flare/organisms/ModalXamanSign/ModalXamanSign';

/**
 * Global QR modal for pending Xaman sign requests outside the connect flow —
 * e.g. the FSA approve payment during a deposit. Renders whenever a flow sets
 * xamanRef on the context; the flow clears it once the payload resolves.
 * Suppressed while connect() runs, because WalletConnectFlow's modal renders
 * those QRs itself.
 */
export function XamanSignModal() {
  const { xamanRef, setXamanRef, connecting } = useXaman();
  const opened = !!xamanRef && !connecting;

  return (
    <Modal
      opened={opened}
      onClose={() => setXamanRef(null)}
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
      <ModalXamanSign
        onClose={() => setXamanRef(null)}
        qrPng={xamanRef?.qrPng}
        deeplink={xamanRef?.deeplink}
        headerLabel="Sign in Xaman"
        primaryText="Scan with the Xaman app"
        secondaryText="Approve the request on your phone to continue"
      />
    </Modal>
  );
}
