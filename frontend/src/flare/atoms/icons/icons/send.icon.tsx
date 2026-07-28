import { BaseIcon, IconProps } from '../icon';

export const Send = (props: IconProps) => (
  <BaseIcon {...props}>
    {(fill) => (
      <path
        d="M0 18V1L20 9.5L0 18ZM1.74302 15.3621L15.5129 9.5L1.74302 3.63793V7.96678L8.04462 9.5L1.74302 11.0332V15.3621Z"
        fill={fill}
      />
    )}
  </BaseIcon>
);
