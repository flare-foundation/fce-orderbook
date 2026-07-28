import { BaseIcon, IconProps } from '../icon';

export const LinkBack = (props: IconProps) => (
  <BaseIcon {...props}>
    {(fill) => (
      <path
        d="M3.83067 9L20 9V11L3.83067 11L11.4257 18.595L10 20L0 10L10 0L11.4257 1.405L3.83067 9Z"
        fill={fill}
      />
    )}
  </BaseIcon>
);
