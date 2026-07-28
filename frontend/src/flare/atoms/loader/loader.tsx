import { Icon } from '../icons';
import { IconProps } from '../icons/icon';
import classes from './loader.module.css';

export const Loader = ({ ...props }: IconProps) => {
  return (
    <Icon.Progress
      size={20}
      className={classes.loader}
      color={!props.variant ? 'var(--mantine-color-token-pairs-charting-turquoise)' : props.color}
      {...props}
    />
  );
};
