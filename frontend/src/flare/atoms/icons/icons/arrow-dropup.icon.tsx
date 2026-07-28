import { BaseIcon, IconProps } from '../icon';

export const ArrowDropup = (props: IconProps) => (
  <BaseIcon {...props}>{(fill) => <path d="M0 15L10 5L20 15H0Z" fill={fill} />}</BaseIcon>
);
