import { Dispatch, SetStateAction, useState } from 'react';
import {
  Group,
  TextInput as MantineTextInput,
  TextInputProps as MantineTextInputProps,
  Stack,
} from '@mantine/core';
import { Loader } from '../../atoms/loader/loader';
import { PrimaryDropdown } from '../../molecules/PrimaryDropdown/PrimaryDropdown';
import { BodyText } from '../../typography/body-text';
import classes from './TextInput.module.css';

export interface TextInputProps extends MantineTextInputProps {
  size?: 'md' | 'lg';
  value: string | number;
  setValue: Dispatch<SetStateAction<string | number | undefined>>;
  placeholder?: string;
  disabled?: boolean;
  error?: string;
  dropDownData?: { value: string; label: string }[];
  dropdownLabel?: string;
  required?: boolean;
  additionalText?: string;
  isLoading?: boolean;
}
export function TextInput({
  value,
  setValue,
  placeholder,
  dropdownLabel,
  dropDownData,
  required,
  additionalText,
  size = 'md',
  isLoading,
  ...rest
}: TextInputProps) {
  const [, setFocus] = useState(false);

  return (
    <Stack gap="xs">
      <div className={classes.inputWrapper}>
        <MantineTextInput
          data-size={size}
          value={value}
          classNames={{
            input: classes.input,
            error: classes.error,
          }}
          styles={{
            input: { paddingRight: dropDownData ? 176 : !required ? 66 : 0 },
            section: { paddingRight: 'var(--mantine-spacing-md)' },
          }}
          onFocus={() => setFocus(true)}
          onBlur={() => setFocus(false)}
          placeholder={placeholder}
          onChange={(event) => setValue && setValue(event.currentTarget.value)}
          rightSectionWidth={dropDownData ? 170 : !required ? 52 : 0}
          rightSectionProps={{
            className: dropDownData ? classes.rightSectionDropdown : classes.rigthSectionRequired,
          }}
          rightSection={
            dropDownData ? (
              <PrimaryDropdown
                placeholder={dropdownLabel || 'Dropdown'}
                onOptionSelect={(val) => setValue && setValue(val)}
                data={dropDownData}
              />
            ) : !required ? (
              <BodyText
                lh="1"
                type="note"
                c={
                  rest.disabled
                    ? 'var(--mantine-color-neutrals-fills-lighter-medium)'
                    : 'var(--mantine-color-neutrals-fills-medium)'
                }
              >
                Optional*
              </BodyText>
            ) : undefined
          }
          {...rest}
          error={additionalText ? false : rest.error}
        />

        {/* vendored tweak: floating placeholder label removed — static input */}
      </div>
      <Group gap="3n" align="center" justify="center">
        {isLoading && <Loader />}
        {additionalText && <BodyText type="sBody">{additionalText}</BodyText>}
      </Group>
    </Stack>
  );
}
