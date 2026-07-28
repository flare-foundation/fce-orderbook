import { Dispatch, SetStateAction } from 'react';
import {
  TextInput as MantineTextInput,
  TextInputProps as MantineTextInputProps,
} from '@mantine/core';
import { Icon } from '../../atoms/icons';
import classes from './SearchInput.module.css';

export interface SearchInputProps extends MantineTextInputProps {
  value: string | number;
  setValue: Dispatch<SetStateAction<string | number | undefined>>;
  placeholder?: string;
}
export function SearchInput({ value, setValue, placeholder, ...rest }: SearchInputProps) {
  return (
    <MantineTextInput
      value={value}
      miw={{ sm: 400 }}
      classNames={{
        input: classes.input,
      }}
      leftSectionWidth={30}
      leftSection={<Icon.Search size={14} variant="gray" />}
      styles={{
        section: { paddingRight: 'var(--mantine-spacing-md)' },
      }}
      placeholder={placeholder}
      onChange={(event) => setValue && setValue(event.currentTarget.value)}
      {...rest}
    />
  );
}
