import { ReactNode } from 'react';
import {
  Group,
  SegmentedControl as MantineSegmentedControl,
  SegmentedControlProps as MantineSegmentedControlProps,
} from '@mantine/core';
import { BodyText } from '../../typography/body-text';
import classes from './SegmentedControl.module.css';

export interface SegmentedControlProps extends Omit<MantineSegmentedControlProps, 'data'> {
  data: { value: string; label: string; leftSection?: ReactNode }[];
}
export const SegmentedControl = ({ ...props }: SegmentedControlProps) => {
  const data = props.data.map((item) => ({
    value: item.value,
    label: (
      <Group gap="8px" wrap="nowrap">
        {item.leftSection}
        <BodyText type="body" c="currentColor">
          {item.label}
        </BodyText>
      </Group>
    ),
  }));
  return (
    <MantineSegmentedControl
      classNames={{
        root: classes.root,
        indicator: classes.indicator,
        label: classes.label,
        control: classes.control,
      }}
      {...props}
      data={data}
    />
  );
};
