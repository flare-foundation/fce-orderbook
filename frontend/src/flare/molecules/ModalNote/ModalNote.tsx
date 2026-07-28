import { ReactNode } from 'react';
import { Box, Collapse, Group } from '@mantine/core';
import { Icon } from '../../atoms/icons';
import { Link } from '../../typography/link';

export interface ModalNoteProps {
  label: string;
  children: ReactNode;
  opened: boolean;
  toggle?: () => void;
}

export const ModalNote = ({ label, children, opened, toggle }: ModalNoteProps) => {
  return (
    <Box mx="auto">
      <Link onClick={toggle} type="smallBodyLink">
        <Group gap="3n">
          {' '}
          {opened ? (
            <Icon.ArrowDropdown size={8} color="var(--mantine-color-pinks-fills-darker)" />
          ) : (
            <Icon.ArrowRight size={8} color="var(--mantine-color-pinks-fills-darker)" />
          )}
          {label}
        </Group>
      </Link>

      <Collapse in={opened}>
        <Box mt="lg">{children}</Box>
      </Collapse>
    </Box>
  );
};
