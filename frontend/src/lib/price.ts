// Must match pricePrecision in internal/extension/handlers.go.
// Allows 3 decimal places of price precision (e.g. 1.001).
export const PRICE_PRECISION = 1000;

export const scalePrice = (humanPrice: number): number =>
  Math.round(humanPrice * PRICE_PRECISION);

export const formatPrice = (rawPrice: number): number =>
  rawPrice / PRICE_PRECISION;
