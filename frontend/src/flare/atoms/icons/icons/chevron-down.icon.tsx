import { BaseIcon, IconProps } from '../icon';

export const ChevronDown = (props: IconProps) => (
  <BaseIcon {...props} viewBox="0 0 22 20">
    {(fill) => (
      <path
        d="M11 16L0.885254 5.8852L2.77046 4L11 12.2296L19.2296 4L21.1148 5.8852L11 16Z"
        fill={fill}
      />
    )}
  </BaseIcon>
);
