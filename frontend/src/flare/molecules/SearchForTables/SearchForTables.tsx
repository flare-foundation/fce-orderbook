import {
  TextInput as MantineTextInput,
  TextInputProps as MantineTextInputProps,
} from '@mantine/core';
import { Icon } from '../../atoms/icons';
import { PrimaryDropdown } from '../../molecules/PrimaryDropdown/PrimaryDropdown';
import classes from './SearchForTables.module.css';

export interface TextInputProps extends MantineTextInputProps {
  size?: 'md' | 'lg';
  value: string;
  setValue: (val: string) => void;
  placeholder?: string;
  disabled?: boolean;
  error?: string;
  dropDownData?: { value: string; label: string }[];
  dropdownLabel?: string;
  required?: boolean;
  additionalText?: string;
  isLoading?: boolean;
  active?: boolean;
}
export function SearchForTables({
  value,
  setValue,
  placeholder = 'Search',
  dropdownLabel,
  dropDownData,
  required,
  additionalText,
  size = 'md',
  active,
  isLoading,
  ...rest
}: TextInputProps) {
  return (
    <MantineTextInput
      data-active={active}
      data-size={size}
      value={value}
      classNames={{
        input: classes.input,
      }}
      styles={{
        input: { paddingRight: dropDownData ? 176 : 0 },
        section: { paddingInline: 'var(--mantine-spacing-md)' },
      }}
      placeholder={placeholder}
      onChange={(event) => setValue && setValue(event.currentTarget.value)}
      rightSectionWidth={dropDownData ? 176 : 0}
      rightSectionProps={{
        className: classes.rightSectionDropdown,
      }}
      rightSection={
        dropDownData ? (
          <PrimaryDropdown
            placeholder={dropdownLabel || 'Dropdown'}
            onOptionSelect={(val) => setValue && setValue(val)}
            data={dropDownData}
          />
        ) : undefined
      }
      leftSection={<Icon.Search variant="gray" size={24} />}
      leftSectionWidth={52}
      {...rest}
      error={additionalText ? false : rest.error}
    />
  );
}
