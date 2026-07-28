import { BaseIcon, IconProps } from '../icon';

export const ChevronRight = (props: IconProps) => (
  <BaseIcon {...props}>
    {(fill) => (
      <path
        d="M12.2046 10L4.06836 1.86381L5.93217 0L15.9322 10L5.93217 20L4.06836 18.1362L12.2046 10Z"
        fill={fill}
      />
    )}
  </BaseIcon>
);
