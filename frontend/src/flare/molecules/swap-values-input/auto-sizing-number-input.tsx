import type { ChangeEvent } from 'react';
import classes from './auto-sizing-number-input.module.css';

interface AutoSizingNumberInputProps {
  value?: number;
  max?: number;
  placeholder?: string;
  onChange?: (value: number) => void;
}

export const AutoSizingNumberInput = ({
  value = 0,
  max,
  placeholder = '0',
  onChange,
}: AutoSizingNumberInputProps) => {
  const displayValue = String(value || '');
  const width = `${Math.max(1, displayValue.length)}ch`;

  const handleChange = (e: ChangeEvent<HTMLInputElement>) => {
    const digitsOnly = e.target.value.replace(/\D/g, '');
    const parsed = digitsOnly ? Number(digitsOnly) : 0;
    const clamped = max !== undefined ? Math.min(parsed, max) : parsed;
    onChange?.(clamped);
  };

  return (
    <input
      type="text"
      inputMode="numeric"
      className={classes.input}
      value={displayValue}
      placeholder={placeholder}
      style={{ width }}
      onChange={handleChange}
    />
  );
};
