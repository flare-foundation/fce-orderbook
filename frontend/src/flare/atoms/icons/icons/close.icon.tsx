import { BaseIcon, IconProps } from '../icon';

export const Close = (props: IconProps) => (
  <BaseIcon {...props}>
    {(fill) => (
      <path
        d="M1.34681 20L0 18.6532L8.65357 10L0 1.34681L1.34681 0L10 8.65358L18.6532 0L20 1.34681L11.3464 10L20 18.6532L18.6532 20L10 11.3464L1.34681 20Z"
        fill={fill}
      />
    )}
  </BaseIcon>
);
