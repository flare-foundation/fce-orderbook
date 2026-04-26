// Must match pricePrecision in internal/extension/handlers.go and
// PricePrecision in tools/pkg/stress/scaling.go.
// 6 decimal places of price precision (e.g. 1.000001) so sub-cent assets like
// FLR (~$0.008) have a meaningful spread; 1 raw tick = 0.000001 quote-per-base.
export const PRICE_PRECISION = 1_000_000;

export const scalePrice = (humanPrice: number): number =>
  Math.round(humanPrice * PRICE_PRECISION);

export const formatPrice = (rawPrice: number): number =>
  rawPrice / PRICE_PRECISION;

// formatHumanAdaptive picks a sensible decimal count based on magnitude of an
// already-human-units price (e.g. chart axis ticks, pre-divided values).
export const formatHumanAdaptive = (v: number): string => {
  if (v >= 100) return v.toFixed(2);
  if (v >= 1) return v.toFixed(4);
  return v.toFixed(6);
};

// formatPriceAdaptive picks a sensible decimal count based on magnitude so
// e.g. BTC at $77,000 shows 2 decimals while FLR at $0.0078 shows 6. Returns
// a string ready for display. Use in place of `.toFixed(3)` everywhere.
export const formatPriceAdaptive = (rawPrice: number): string =>
  formatHumanAdaptive(formatPrice(rawPrice));
