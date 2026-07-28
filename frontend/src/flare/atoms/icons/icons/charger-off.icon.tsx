import { BaseIcon, IconProps } from '../icon';

export const ChargerOff = (props: IconProps) => (
  <BaseIcon {...props}>
    {(fill) => (
      <path
        d="M10 0C15.5228 0 20 4.47715 20 10C20 15.5228 15.5228 20 10 20C4.47715 20 0 15.5228 0 10C0 4.47715 4.47715 0 10 0ZM10 2C5.58172 2 2 5.58172 2 10C2 14.4183 5.58172 18 10 18C14.4183 18 18 14.4183 18 10C18 5.58172 14.4183 2 10 2ZM14 10.75H6V9.25H14V10.75Z"
        fill={fill}
      />
    )}
  </BaseIcon>
);
