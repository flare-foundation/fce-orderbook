import { Box, Group, Stack } from '@mantine/core';
import { Button } from '../../atoms/button/button';
import { type BaseTokenProps } from '../../atoms/token-icon/token-icon';
import { ModalHeader } from '../../molecules/ModalHeader/ModalHeader';
import { BodyText } from '../../typography/body-text';
import classes from './ModalSuccess.module.css';

export interface ModalSuccess {
  onClose: () => void;
  ContentIcon: React.ComponentType<BaseTokenProps>;
  headerText: string;
  onClick?: () => void;
  buttonText: string;
  LeftIcon?: React.ComponentType<BaseTokenProps>;
  leftText?: string;
  contentPrimaryText: string;
  contentSecondaryText?: string;
}
export const ModalSuccess = ({
  onClose,
  headerText,
  onClick,
  buttonText,
  LeftIcon,
  leftText,
  ContentIcon,
  contentPrimaryText,
  contentSecondaryText,
}: ModalSuccess) => {
  return (
    <Box className={classes.wrapper}>
      <Stack px="xl" py="lg" gap="xl">
        <ModalHeader label={headerText} onClose={onClose} />
        <Group gap="sm">
          {LeftIcon && <LeftIcon size={42} />}
          {leftText && (
            <BodyText type="sBody" c="var(--mantine-color-neutrals-gray-500)">
              {leftText}
            </BodyText>
          )}
        </Group>
      </Stack>

      <Stack gap="md" px="xl" align="center">
        <Box py="xl2">
          <ContentIcon size={120} />
        </Box>
        <Stack gap="2n" align="center" pb="xl">
          <BodyText type="lBody">{contentPrimaryText}</BodyText>
          <BodyText c="var(--mantine-color-neutrals-fills-medium)">
            {contentSecondaryText}{' '}
          </BodyText>
        </Stack>
      </Stack>
      <Stack align="center" px="xl" pb="xl2">
        <Button variant="secondary" onClick={onClick}>
          {buttonText}
        </Button>
      </Stack>
    </Box>
  );
};
