import { BaseIcon, IconProps } from '../icon';

export const LinkUp = (props: IconProps) => (
  <BaseIcon {...props}>
    {(fill) => (
      <path
        d="M9 20V3.83067L1.405 11.4257L0 10L10 0L20 10L18.595 11.4257L11 3.83067V20H9Z"
        fill={fill}
      />
    )}
  </BaseIcon>
);
