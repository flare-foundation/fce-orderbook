import { BaseIcon, IconProps } from '../icon';

export const LinkExternal = (props: IconProps) => (
  <BaseIcon {...props}>
    {(fill) => (
      <path
        d="M1.42048 20L0 18.5795L16.5247 2.03981H1.3312V0H20V18.6692H17.9606V3.47572L1.42048 20Z"
        fill={fill}
      />
    )}
  </BaseIcon>
);
