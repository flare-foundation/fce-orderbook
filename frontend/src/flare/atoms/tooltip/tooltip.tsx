import { ReactNode } from 'react';
import { Box, Tooltip as MantineTooltip, TooltipProps as MantineTooltipProps } from '@mantine/core';
import classes from './tooltip.module.css';

export interface TooltipProps extends MantineTooltipProps {
  label: ReactNode;
  children: ReactNode;
}

export function Tooltip({ label, children, ...props }: TooltipProps) {
  return (
    <MantineTooltip
      arrowSize={18}
      withArrow
      label={label}
      classNames={{ arrow: classes.arrow, tooltip: classes.tooltip }}
      maw={235}
      multiline
      {...props}
    >
      <Box>{children}</Box>
    </MantineTooltip>
  );
}
