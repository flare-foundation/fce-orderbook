import { BaseIcon, IconProps } from '../icon';

export const X = (props: IconProps) => (
  <BaseIcon {...props} viewBox="0 0 18 18" variant="gray">
    {(fill) => (
      <>
        <rect x="0.5" y="0.5" width="17" height="17" rx="8.5" fill={fill} />
        <path
          d="M4.52195 5L7.9971 9.41303L4.5 13H5.2875L8.34832 9.85863L10.822 13H13.5L9.83003 8.33941L13.0843 5H12.2982L9.47881 7.89251L7.20137 5H4.52195ZM5.67988 5.54984H6.91052L12.3434 12.4489H11.1128L5.67988 5.54984Z"
          fill="white"
        />
      </>
    )}
  </BaseIcon>
);
