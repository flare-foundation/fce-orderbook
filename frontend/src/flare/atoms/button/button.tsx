import { forwardRef, type ReactNode } from 'react';
import clsx from 'clsx';
import {
  Button as ButtonMantine,
  createPolymorphicComponent,
  type ButtonProps as MantineButtonProps,
} from '@mantine/core';
import classes from './button.module.css';

interface ButtonProps extends MantineButtonProps {
  variant?: 'primary' | 'secondary' | 'tertiary';
  size?: 'lg' | 'md';
  children?: ReactNode;
}

const _Button = forwardRef<HTMLButtonElement, ButtonProps>(
  ({ variant = 'primary', size = 'md', children, className, ...rest }, ref) => {
    return (
      <ButtonMantine
        ref={ref}
        className={clsx(classes[variant], className)}
        fz={size === 'lg' ? 'body' : 'sBody'}
        {...rest}
      >
        {children}
      </ButtonMantine>
    );
  }
);

export const Button = createPolymorphicComponent<'button', ButtonProps>(_Button);
