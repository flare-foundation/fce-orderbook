import { BaseIcon, IconProps } from '../icon';

export const ChevronBack = (props: IconProps) => (
  <BaseIcon {...props}>
    {(fill) => (
      <path
        d="M14.0679 20L4.06787 10L14.0679 0L15.9317 1.86381L7.79548 10L15.9317 18.1362L14.0679 20Z"
        fill={fill}
      />
    )}
  </BaseIcon>
);
