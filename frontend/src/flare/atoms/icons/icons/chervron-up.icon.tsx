import { BaseIcon, IconProps } from '../icon';

export const ChevronUp = (props: IconProps) => (
  <BaseIcon {...props}>
    {(fill) => (
      <path
        d="M10 7.79598L1.86381 15.9322L0 14.0684L10 4.06836L20 14.0684L18.1362 15.9322L10 7.79598Z"
        fill={fill}
      />
    )}
  </BaseIcon>
);
