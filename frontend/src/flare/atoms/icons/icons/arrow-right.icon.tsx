import { BaseIcon, IconProps } from '../icon';

export const ArrowRight = (props: IconProps) => (
  <BaseIcon {...props}>{(fill) => <path d="M5 20V0L15 10L5 20Z" fill={fill} />}</BaseIcon>
);
