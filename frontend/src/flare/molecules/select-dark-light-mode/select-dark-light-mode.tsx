import {
  Box,
  SegmentedControl,
  useComputedColorScheme,
  useMantineColorScheme,
} from '@mantine/core';
import { ActionIcon } from '../../atoms/action-icon/action-icon';
import { Icon } from '../../atoms/icons';
import classes from './select-dark-light-mode.module.css';

export type ColorScheme = 'dark' | 'light';

export interface SelectDarkLightModeProps {
  variant: 'single' | 'dual';
}

export const SelectDarkLightMode = ({ variant }: SelectDarkLightModeProps) => {
  const { setColorScheme } = useMantineColorScheme();
  const computedColorScheme = useComputedColorScheme('light', {
    getInitialValueInEffect: true,
  });

  if (variant === 'single') {
    return (
      <ActionIcon
        onClick={() => setColorScheme(computedColorScheme === 'light' ? 'dark' : 'light')}
        aria-label="Toggle color scheme"
        className={classes.icon}
      >
        <Box darkHidden>
          <Icon.Dark size={20} color="currentColor" />
        </Box>
        <Box lightHidden>
          <Icon.Light size={20} color="currentColor" />
        </Box>
      </ActionIcon>
    );
  }

  return (
    <SegmentedControl
      classNames={{
        root: classes.root,
        indicator: classes.indicator,
        label: classes.label,
      }}
      onChange={(val) => setColorScheme(val as ColorScheme)}
      value={computedColorScheme}
      data={[
        { value: 'dark', label: <Icon.Dark size={20} color="currentColor" /> },
        { value: 'light', label: <Icon.Light size={20} color="currentColor" /> },
      ]}
    />
  );
};
