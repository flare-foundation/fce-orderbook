import { useEffect, useMemo, useRef } from 'react';
import {
  createChart,
  CandlestickSeries,
  CrosshairMode,
  type IChartApi,
  type ISeriesApi,
  type UTCTimestamp,
} from 'lightweight-charts';
import { useBookState } from '../hooks/useBookState';
import { bucketMatches, type Timeframe, TF_SECONDS } from '../lib/candles';

interface ChartProps {
  pair: string;
  timeframe: Timeframe;
}

function cssVar(name: string, fallback: string): string {
  if (typeof window === 'undefined') return fallback;
  const v = getComputedStyle(document.documentElement).getPropertyValue(name).trim();
  return v || fallback;
}

export function Chart({ pair, timeframe }: ChartProps) {
  const { matches } = useBookState(pair);
  const containerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const seriesRef = useRef<ISeriesApi<'Candlestick'> | null>(null);
  const lastTfRef = useRef<Timeframe | null>(null);

  const candles = useMemo(() => bucketMatches(matches, timeframe), [matches, timeframe]);

  useEffect(() => {
    if (!containerRef.current) return;

    const fgMute = cssVar('--fg-mute', '#5a5a54');
    const line = cssVar('--line', '#1f1f1f');
    const bid = cssVar('--bid', '#26a269');
    const ask = cssVar('--ask', '#c01c28');
    const accent = cssVar('--accent', '#c96aa8');

    const chart = createChart(containerRef.current, {
      width: containerRef.current.clientWidth,
      height: containerRef.current.clientHeight,
      layout: {
        background: { color: 'transparent' },
        textColor: fgMute,
        fontFamily: 'var(--f-mono, ui-monospace, monospace)',
        fontSize: 10,
      },
      grid: {
        vertLines: { color: line },
        horzLines: { color: line },
      },
      rightPriceScale: {
        borderColor: line,
        scaleMargins: { top: 0.1, bottom: 0.1 },
      },
      timeScale: {
        borderColor: line,
        timeVisible: true,
        secondsVisible: false,
      },
      crosshair: {
        mode: CrosshairMode.Normal,
        vertLine: { color: fgMute, width: 1, style: 2, labelBackgroundColor: accent },
        horzLine: { color: fgMute, width: 1, style: 2, labelBackgroundColor: accent },
      },
      localization: { priceFormatter: (p: number) => p.toFixed(3) },
    });

    const series = chart.addSeries(CandlestickSeries, {
      upColor: bid,
      downColor: ask,
      wickUpColor: bid,
      wickDownColor: ask,
      borderVisible: false,
      priceFormat: { type: 'price', precision: 3, minMove: 0.001 },
    });

    chartRef.current = chart;
    seriesRef.current = series;

    const ro = new ResizeObserver(() => {
      if (!containerRef.current || !chartRef.current) return;
      chartRef.current.applyOptions({
        width: containerRef.current.clientWidth,
        height: containerRef.current.clientHeight,
      });
    });
    ro.observe(containerRef.current);

    return () => {
      ro.disconnect();
      chart.remove();
      chartRef.current = null;
      seriesRef.current = null;
      lastTfRef.current = null;
    };
  }, []);

  useEffect(() => {
    const series = seriesRef.current;
    const chart = chartRef.current;
    if (!series || !chart) return;

    const data = candles.map(c => ({
      time: c.time as UTCTimestamp,
      open: c.open,
      high: c.high,
      low: c.low,
      close: c.close,
    }));

    series.setData(data);

    if (lastTfRef.current !== timeframe) {
      const last = data[data.length - 1]?.time;
      if (typeof last === 'number') {
        const span = Math.max(60, TF_SECONDS[timeframe] * 60);
        chart.timeScale().setVisibleRange({
          from: (last - span) as UTCTimestamp,
          to: (last + TF_SECONDS[timeframe]) as UTCTimestamp,
        });
      } else {
        chart.timeScale().fitContent();
      }
      lastTfRef.current = timeframe;
    }
  }, [candles, timeframe]);

  const hasData = candles.length > 0;

  return (
    <div style={{ position: 'relative', width: '100%', height: '100%' }}>
      <div ref={containerRef} style={{ width: '100%', height: '100%' }} />
      {!hasData && (
        <div
          style={{
            position: 'absolute',
            inset: 0,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            pointerEvents: 'none',
          }}
        >
          <div className="empty-hint">NO TRADES · WAITING FOR MARKET DATA</div>
        </div>
      )}
    </div>
  );
}
