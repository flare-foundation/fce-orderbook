import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { MantineProvider } from "@mantine/core";
import "@mantine/core/styles.css";
// Flare Design System: Satoshi font, color/size tokens, and the Mantine theme
// that binds them (used by the wallet-connect modals).
import "./flare/satoshi.css";
import "./flare/global.css";
import { theme } from "./flare/theme";
import App from "./App";
import "./index.css";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <MantineProvider theme={theme} defaultColorScheme="dark">
      <App />
    </MantineProvider>
  </StrictMode>
);
