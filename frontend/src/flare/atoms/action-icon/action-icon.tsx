import { forwardRef } from 'react';
import clsx from 'clsx';
import {
  createPolymorphicComponent,
  UnstyledButton as MantineUnstyledButton,
  type UnstyledButtonProps as MantineUnstyledButtonProps,
} from '@mantine/core';
import classes from './action-icon.module.css';

export interface ActionIconProps extends MantineUnstyledButtonProps {}

const _ActionIcon = forwardRef<HTMLButtonElement, ActionIconProps>(
  ({ className, ...props }, ref) => {
    return (
      <MantineUnstyledButton ref={ref} className={clsx(classes.actionIcon, className)} {...props} />
    );
  }
);

export const ActionIcon = createPolymorphicComponent<'button', ActionIconProps>(_ActionIcon);
