import { BaseIcon, IconProps } from '../icon';

export const Circle = (props: IconProps) => (
  <BaseIcon {...props}>{(fill) => <circle cx="10" cy="10" r="10" fill={fill} />}</BaseIcon>
);
