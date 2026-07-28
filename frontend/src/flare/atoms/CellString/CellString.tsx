import { BodyText } from '../../typography/body-text';

export interface CellStringProps {
  label: string;
  color?: string;
}

export const CellString = ({ label, color }: CellStringProps) => (
  <BodyText
    c={color || 'var(--mantine-color-neutrals-fills-medium)'}
    fz={{ base: 'sBody', sm: 'body' }}
    lh={{ base: 'sBody', sm: 'body' }}
  >
    {label}
  </BodyText>
);
