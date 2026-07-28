import { BaseIcon, IconProps } from '../icon';

export const Menu = (props: IconProps) => (
  <BaseIcon {...props}>
    {(fill) => (
      <path
        d="M0 17V15.1368H20V17H0ZM0 10.9318V9.06824H20V10.9318H0ZM0 4.86321V3H20V4.86321H0Z"
        fill={fill}
      />
    )}
  </BaseIcon>
);
