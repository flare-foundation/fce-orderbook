import { ReactNode } from 'react';
import { Box } from '@mantine/core';
import classes from './error-box.module.css';

export interface ErrorBoxProps {
  description: ReactNode;
}

export const ErrorBox = ({ description }: ErrorBoxProps) => {
  return <Box className={classes.errorBox}>{description}</Box>;
};
