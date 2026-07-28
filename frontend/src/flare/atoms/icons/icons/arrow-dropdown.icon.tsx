import { BaseIcon, IconProps } from '../icon';

export const ArrowDropdown = (props: IconProps) => (
  <BaseIcon {...props}>{(fill) => <path d="M10 15L0 5H20L10 15Z" fill={fill} />}</BaseIcon>
);
