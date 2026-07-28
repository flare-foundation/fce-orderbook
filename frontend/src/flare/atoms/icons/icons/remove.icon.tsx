import { BaseIcon, IconProps } from '../icon';

export const Remove = (props: IconProps) => (
  <BaseIcon {...props}>{(fill) => <path d="M0 11V9H20V11H0Z" fill={fill} />}</BaseIcon>
);
