import { BaseIcon, IconProps } from '../icon';

export const Check = (props: IconProps) => (
  <BaseIcon {...props}>
    {(fill) => (
      <path
        d="M6.29151 17L0 10.7085L1.80281 8.90568L6.29151 13.3944L17.6859 2L19.4887 3.80281L6.29151 17Z"
        fill={fill}
      />
    )}
  </BaseIcon>
);
