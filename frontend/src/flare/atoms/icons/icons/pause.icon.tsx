import { BaseIcon, IconProps } from '../icon';

export const Pause = (props: IconProps) => (
  <BaseIcon {...props}>
    {(fill) => (
      <path
        d="M11.9231 20V0H20V20H11.9231ZM0 20V0H8.07692V20H0ZM14.2308 17.6923H17.6923V2.30769H14.2308V17.6923ZM2.30769 17.6923H5.76923V2.30769H2.30769V17.6923Z"
        fill={fill}
      />
    )}
  </BaseIcon>
);
