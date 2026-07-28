import { JSX } from 'react';
import { Box, Group } from '@mantine/core';
import { BodyText } from '../../typography/body-text';
import { BaseIconProps } from '../PrimaryDropdown/PrimaryDropdown';

export interface SearchResultProps {
  label: string;
  Icon?: (props: BaseIconProps) => JSX.Element;
  additionalInfo?: string;
}

export const SearchResult = ({ label, Icon, additionalInfo }: SearchResultProps) => (
  <Group justify="space-between" wrap="nowrap" bg="var">
    <Group gap="xs">
      {Icon && <Icon size={42} radius={8} />}
      <Box c="currentColor" fz="body" lh="body" fw="500">
        {label}
      </Box>
    </Group>
    {additionalInfo && (
      <BodyText c="var(--mantine-color-neutrals-fills-medium)">{additionalInfo}</BodyText>
    )}
  </Group>
);
