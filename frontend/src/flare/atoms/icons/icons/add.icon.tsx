import { BaseIcon, IconProps } from '../icon';

export const Add = (props: IconProps) => (
  <BaseIcon {...props}>
    {(fill) => (
      <path
        d="M9.11765 20V10.8824H0V9.11765H9.11765V0H10.8824V9.11765H20V10.8824H10.8824V20H9.11765Z"
        fill={fill}
      />
    )}
  </BaseIcon>
);
