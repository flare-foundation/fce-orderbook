import { Box } from '@mantine/core';
import classes from './modal-pill.module.css';

export interface ModalPill {
  label: string;
  onClick?: () => void;
}
export const ModalPill = ({ label, onClick }: ModalPill) => {
  return (
    <Box onClick={onClick} style={{ cursor: onClick ? 'pointer' : 'auto' }} className={classes.box}>
      {label}
    </Box>
  );
};
