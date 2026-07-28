import { BaseIcon, IconProps } from '../icon';

export const ArrowLeft = (props: IconProps) => (
  <BaseIcon {...props}>{(fill) => <path d="M15 20L5 10L15 0V20Z" fill={fill} />}</BaseIcon>
);
