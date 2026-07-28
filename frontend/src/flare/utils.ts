// Vendored from flare_design_system utils.ts — only what our vendored
// components import.
export function truncateString(address: string, start: number, end: number) {
  return address ? `${address.substring(0, start)}...${end === 0 ? '' : address.slice(-end)}` : '';
}
