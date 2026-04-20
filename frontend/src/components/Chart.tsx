import { useState, useRef, useMemo } from 'react';
import { useBookState } from '../hooks/useBookState';
import { formatPrice } from '../lib/price';

interface ChartProps {
  pair: string;
}

const W = 680, H = 180, PAD_L = 48, PAD_R = 8, PAD_T = 14, PAD_B = 18;

export function Chart({ pair }: ChartProps) {
  const { matches } = useBookState(pair);

  const history = useMemo(() => {
    if (!matches.length) return [];
    return [...matches]
      .sort((a, b) => {
        const ta = a.timestamp > 1e12 ? a.timestamp : a.timestamp * 1000;
        const tb = b.timestamp > 1e12 ? b.timestamp : b.timestamp * 1000;
        return ta - tb;
      })
      .map(m => ({ p: formatPrice(m.price) }));
  }, [matches]);

  const chartData = useMemo(() => {
    if (history.length < 2) return null;
    const prices = history.map(h => h.p);
    let min = Math.min(...prices), max = Math.max(...prices);
    if (max - min < 0.01) { max += 0.01; min -= 0.01; }
    const pad = (max - min) * 0.12;
    min -= pad; max += pad;
    const iw = W - PAD_L - PAD_R, ih = H - PAD_T - PAD_B;
    const points = history.map((d, i) => ({
      x: PAD_L + (i / (history.length - 1)) * iw,
      y: PAD_T + ih - ((d.p - min) / (max - min)) * ih,
      p: d.p,
    }));
    const path = points.map((p, i) => (i === 0 ? 'M' : 'L') + p.x.toFixed(1) + ' ' + p.y.toFixed(1)).join(' ');
    const last = points[points.length - 1];
    const areaPath = path + ` L ${last.x.toFixed(1)} ${(H - PAD_B).toFixed(1)} L ${points[0].x.toFixed(1)} ${(H - PAD_B).toFixed(1)} Z`;

    const yTicks: { p: number; y: number }[] = [];
    for (let i = 0; i <= 4; i++) {
      const frac = i / 4;
      yTicks.push({ p: min + (max - min) * (1 - frac), y: PAD_T + ih * frac });
    }
    return { points, path, areaPath, yTicks, lastY: last.y };
  }, [history]);

  const mid = history[history.length - 1]?.p ?? 0;
  const up = history.length >= 2 ? mid >= history[0].p : true;

  const [hover, setHover] = useState<{ x: number; y: number; p: number } | null>(null);
  const svgRef = useRef<SVGSVGElement>(null);

  const onMove = (e: React.MouseEvent<SVGSVGElement>) => {
    if (!chartData || !svgRef.current) return;
    const rect = svgRef.current.getBoundingClientRect();
    const xRel = (e.clientX - rect.left) / rect.width;
    const svgX = xRel * W;
    if (svgX < PAD_L || svgX > W - PAD_R) { setHover(null); return; }
    const frac = (svgX - PAD_L) / (W - PAD_L - PAD_R);
    const idx = Math.max(0, Math.min(chartData.points.length - 1, Math.round(frac * (chartData.points.length - 1))));
    setHover(chartData.points[idx]);
  };

  if (history.length < 2) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%' }}>
        <div className="empty-hint">NO TRADES · WAITING FOR MARKET DATA</div>
      </div>
    );
  }

  return (
    <svg
      ref={svgRef}
      className="chart"
      viewBox={`0 0 ${W} ${H}`}
      preserveAspectRatio="none"
      onMouseMove={onMove}
      onMouseLeave={() => setHover(null)}
    >
      <defs>
        <linearGradient id="chart-fill" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor={up ? 'var(--bid)' : 'var(--ask)'} stopOpacity="0.18" />
          <stop offset="100%" stopColor={up ? 'var(--bid)' : 'var(--ask)'} stopOpacity="0" />
        </linearGradient>
      </defs>

      {chartData!.yTicks.map((t, i) => (
        <g key={i}>
          <line x1={PAD_L} x2={W - PAD_R} y1={t.y} y2={t.y} className="chart-grid" />
          <text x={PAD_L - 6} y={t.y + 3} className="chart-tick" textAnchor="end">
            {t.p.toFixed(3)}
          </text>
        </g>
      ))}

      <path d={chartData!.areaPath} fill="url(#chart-fill)" />
      <path d={chartData!.path} fill="none" stroke={up ? 'var(--bid)' : 'var(--ask)'} strokeWidth="1.25" />

      <line x1={PAD_L} x2={W - PAD_R} y1={chartData!.lastY} y2={chartData!.lastY} className="chart-last" />
      <rect x={W - PAD_R - 60} y={chartData!.lastY - 8} width="60" height="14" className="chart-last-bg" />
      <text x={W - PAD_R - 4} y={chartData!.lastY + 3} className="chart-last-tx" textAnchor="end">
        {mid.toFixed(3)}
      </text>

      {hover && (
        <>
          <line x1={hover.x} x2={hover.x} y1={PAD_T} y2={H - PAD_B} className="chart-ch" />
          <line x1={PAD_L} x2={W - PAD_R} y1={hover.y} y2={hover.y} className="chart-ch" />
          <circle cx={hover.x} cy={hover.y} r="2.5" fill="var(--accent)" />
          <rect x={hover.x + 6} y={hover.y - 18} width="76" height="14" className="chart-ch-bg" />
          <text x={hover.x + 10} y={hover.y - 7} className="chart-ch-tx">
            {hover.p.toFixed(3)}
          </text>
        </>
      )}
    </svg>
  );
}
