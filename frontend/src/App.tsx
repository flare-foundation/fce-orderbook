import "@rainbow-me/rainbowkit/styles.css";

import { getDefaultConfig, RainbowKitProvider, darkTheme } from "@rainbow-me/rainbowkit";
import { WagmiProvider } from "wagmi";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { coston2 } from "./config/chain";
import { env } from "./config/env";
import { ToastProvider } from "./components/ui/Toast";
import { Trade } from "./pages/Trade";

const config = getDefaultConfig({
  appName: "Orderbook",
  projectId: env.walletConnectProjectId || "placeholder-project-id",
  chains: [coston2],
});

const queryClient = new QueryClient();

export default function App() {
  return (
    <WagmiProvider config={config}>
      <QueryClientProvider client={queryClient}>
        <RainbowKitProvider theme={darkTheme()}>
          <ToastProvider>
            <Trade />
          </ToastProvider>
        </RainbowKitProvider>
      </QueryClientProvider>
    </WagmiProvider>
  );
}
