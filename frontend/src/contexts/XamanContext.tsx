// XamanContext — connection state for the Xaman (XRPL) wallet path.
//
// Connecting resolves the user's deterministic PersonalAccount (PA) on Flare
// from their XRPL r-address via the MasterAccountController; the PA is then the
// user's identity everywhere (orders, TEE balances, deposits). Because the PA
// has no private key, a local session key signs authenticated ops; it is bound
// to the PA in the TEE via an in-enclave-verified XRPL signature
// (BIND_SESSION_SIG) as part of connecting — free, no gas, no FDC wait.
//
// The app always starts disconnected (mirrors shielded-transfer): reconnecting
// is one Xaman scan, and a returning user's binding is detected so the bind
// scan is skipped.

import { createContext, useCallback, useContext, useState, type ReactNode } from "react";
import type { Address } from "viem";
import { personalAccountFor } from "../lib/fsa";
import { readClient } from "../lib/readClient";
import { createPayload, pollPayload, type XamanPayloadRef } from "../lib/xaman";
import { resolveSession, type Session } from "../lib/session";
import { bindSessionKeySig, checkBinding } from "../lib/fsaTee";
import { INSTRUCTION_SENDER } from "../config/addresses";
import { coston2 } from "../config/chain";

interface XamanState {
  /** XRPL r-address of the connected Xaman account (null = not connected). */
  xrplAddress: string | null;
  /** The PersonalAccount — the user's EVM identity while Xaman is connected. */
  pa: Address | null;
  /** Session key bound to the PA (signs WITHDRAW_REQUEST). */
  session: Session | null;
  connecting: boolean;
  /** Sign-request payload (QR + deeplink) currently awaiting approval in Xaman. */
  xamanRef: XamanPayloadRef | null;
  setXamanRef: (ref: XamanPayloadRef | null) => void;
  connect: () => Promise<void>;
  disconnect: () => void;
}

const XamanContext = createContext<XamanState | null>(null);

export function XamanProvider({ children }: { children: ReactNode }) {
  const [xrplAddress, setXrplAddress] = useState<string | null>(null);
  const [pa, setPa] = useState<Address | null>(null);
  const [session, setSession] = useState<Session | null>(null);
  const [connecting, setConnecting] = useState(false);
  const [xamanRef, setXamanRef] = useState<XamanPayloadRef | null>(null);

  const connect = useCallback(async () => {
    if (connecting) return;
    setConnecting(true);
    try {
      // 1. SignIn in Xaman → r-address + signed blob (session-key derivation seed).
      const r = await createPayload("/login", { memo: "flare orderbook: connect" });
      setXamanRef(r);
      let login: { account: string; hex: string | null };
      try {
        login = await pollPayload(r.uuid, (p) => (p.account ? { account: p.account, hex: p.hex } : null));
      } finally {
        setXamanRef(null);
      }

      // 2. Deterministic PersonalAccount for that r-address.
      const account = await personalAccountFor(login.account, readClient());

      // 3. Pick the session key (derived-from-signature preferred) and make
      //    sure the TEE holds a binding for it — bind if not (one more scan).
      const binding = await checkBinding(account).catch(() => null);
      const { session: sess } = resolveSession(
        { chainId: coston2.id, contract: INSTRUCTION_SENDER, xrplAddress: login.account },
        login.hex,
        binding?.bound && binding.sessionPub ? binding.sessionPub : null,
      );
      const alreadyBound =
        !!binding?.bound &&
        binding.sessionPub?.toLowerCase() === sess.sessionPubHex.toLowerCase();
      if (!alreadyBound) {
        await bindSessionKeySig({ session: sess, onXamanRef: setXamanRef });
      }

      setXrplAddress(login.account);
      setPa(account);
      setSession(sess);
    } finally {
      setConnecting(false);
    }
  }, [connecting]);

  const disconnect = useCallback(() => {
    setXrplAddress(null);
    setPa(null);
    setSession(null);
    setXamanRef(null);
  }, []);

  return (
    <XamanContext.Provider
      value={{ xrplAddress, pa, session, connecting, xamanRef, setXamanRef, connect, disconnect }}
    >
      {children}
    </XamanContext.Provider>
  );
}

export function useXaman(): XamanState {
  const ctx = useContext(XamanContext);
  if (!ctx) throw new Error("useXaman must be used inside <XamanProvider>");
  return ctx;
}
