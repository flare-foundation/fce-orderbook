import { Group } from '@mantine/core';
import { CellString } from '../../atoms/CellString/CellString';
import { CopyButton } from '../../atoms/copy-button/copy-button';
import { truncateString } from '../../utils';

export interface StringWithActionProps {
  label: string;
  shorten?: boolean;
  color?: string;
}

export const StringWithAction = ({ label, shorten = true, color }: StringWithActionProps) => {
  return (
    <Group gap="xs">
      <CellString color={color} label={shorten ? truncateString(label, 5, 5) : label} />
      <CopyButton value={label} />
    </Group>
  );
};
