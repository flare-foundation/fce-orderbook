interface Props {
  pairs: readonly string[];
  selected: string;
  onChange: (pair: string) => void;
}

export function PairSelector({ pairs, selected, onChange }: Props) {
  if (pairs.length <= 1) {
    return <span className="text-sm font-medium text-gray-300">{selected}</span>;
  }

  return (
    <select
      value={selected}
      onChange={(e) => onChange(e.target.value)}
      className="bg-gray-800 border border-gray-700 rounded-lg px-3 py-1.5 text-sm text-gray-100 focus:outline-none focus:border-blue-500"
    >
      {pairs.map((pair) => (
        <option key={pair} value={pair}>
          {pair}
        </option>
      ))}
    </select>
  );
}
