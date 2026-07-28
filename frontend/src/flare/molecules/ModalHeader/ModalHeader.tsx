import { Group } from '@mantine/core';
import { ActionIcon } from '../../atoms/action-icon/action-icon';
import { Icon } from '../../atoms/icons';
import { Heading } from '../../typography/heading';

export interface ModalHeaderProps {
  label: string;
  onClose?: () => void;
  onBackClick?: () => void;
  withCloseButton?: boolean;
  withBackIcon?: boolean;
}

export const ModalHeader = ({
  label,
  withBackIcon,
  onClose,
  onBackClick,
  withCloseButton = true,
}: ModalHeaderProps) => {
  return (
    <Group justify="space-between">
      <Group gap="md">
        {withBackIcon && (
          <ActionIcon onClick={onBackClick}>
            <Icon.LinkBack size={24} />
          </ActionIcon>
        )}
        <Heading order={6} c="var(--mantine-color-neutrals-fills-dark)">
          {label}
        </Heading>
      </Group>
      {withCloseButton && (
        <ActionIcon onClick={onClose} p="10px">
          <Icon.Close size={24} />
        </ActionIcon>
      )}
    </Group>
  );
};
