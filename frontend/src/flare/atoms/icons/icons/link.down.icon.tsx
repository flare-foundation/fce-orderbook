import { BaseIcon, IconProps } from '../icon';

export const LinkDown = (props: IconProps) => (
  <BaseIcon {...props}>
    {(fill) => (
      <path
        d="M9 0V16.1693L1.405 8.57433L0 10L10 20L20 10L18.595 8.57433L11 16.1693V0H9Z"
        fill={fill}
      />
    )}
  </BaseIcon>
);
