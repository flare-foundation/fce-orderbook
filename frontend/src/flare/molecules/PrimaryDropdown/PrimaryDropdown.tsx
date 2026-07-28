import { ReactNode, useState } from 'react';
import {
  Combobox,
  Group,
  MantineStyleProp,
  UnstyledButton,
  UnstyledButtonProps,
  useCombobox,
} from '@mantine/core';
import { Icon } from '../../atoms/icons';
import { BodyText } from '../../typography/body-text';
import { Heading } from '../../typography/heading';
import { SearchForTables } from '../SearchForTables/SearchForTables';
import { SearchResult, SearchResultProps } from '../SearchResult/SearchResult';
import classes from './PrimaryDropdown.module.css';

export interface BaseIconProps {
  size?: number;
  className?: string;
  color?: string;
  radius?: number;
}
export interface DropdownData<T extends string | number> extends SearchResultProps {
  value: T;
}

export interface PrimaryDropdownProps<T extends string | number> extends UnstyledButtonProps {
  size?: 'md' | 'lg' | 'xl';
  value?: T;
  onOptionSelect?: (value: T) => void;
  data: DropdownData<T>[];
  placeholder?: string;
  disabled?: boolean;
  rightIcon?: ReactNode;
  onOpen?: () => void;
  onClose?: () => void;
  appliedOptionStyle?: boolean;
  leftSection?: ReactNode;
  buttonStyle?: MantineStyleProp;
  withSearch?: boolean;
  searchPlaceholder?: string;
}
export const PrimaryDropdown = <T extends string | number>({
  data,
  size = 'md',
  value,
  placeholder = 'Select value',
  onOptionSelect,
  disabled,
  rightIcon,
  onClose,
  onOpen,
  appliedOptionStyle = true,
  buttonStyle,
  searchPlaceholder,
  withSearch = false,
  ...rest
}: PrimaryDropdownProps<T>) => {
  const [search, setSearch] = useState('');

  const [isHovered, setIsHovered] = useState(false);
  const combobox = useCombobox({
    onDropdownClose: () => combobox.resetSelectedOption(),
  });
  const options = data
    .filter(
      (item) =>
        item.label.toLowerCase().includes(search.toLowerCase().trim()) ||
        item.additionalInfo?.toLowerCase().includes(search.toLowerCase().trim())
    )
    .map((item, index) => {
      return (
        <Combobox.Option
          value={item.value.toString()}
          key={index}
          className={classes.option}
          data-size={size}
        >
          <SearchResult {...item} />
        </Combobox.Option>
      );
    });
  const selectedOption = data.find((item) => item.value === value);
  return (
    <div onMouseEnter={() => setIsHovered(true)} onMouseLeave={() => setIsHovered(false)}>
      <Combobox
        attributes={{ dropdown: { 'data-selected': true } }}
        middlewares={{ flip: false, shift: false }}
        store={combobox}
        onOptionSubmit={(val) => {
          onOptionSelect &&
            onOptionSelect(typeof val === 'string' ? (val.toString() as T) : (Number(val) as T));
          combobox.closeDropdown();
        }}
        offset={0}
        disabled={disabled}
        withinPortal
        classNames={{ dropdown: classes.dropdown }}
        onOpen={onOpen}
        onClose={onClose}
      >
        <Combobox.Target>
          <UnstyledButton
            className={classes.unstyledButton}
            disabled={disabled}
            style={buttonStyle}
            data-hovered={!value && isHovered && !disabled}
            data-disabled={disabled}
            data-opened={combobox.dropdownOpened}
            data-applied={appliedOptionStyle && value && !disabled}
            data-nothovered={(!value || !appliedOptionStyle) && !isHovered && !disabled}
            data-size={size}
            onClick={() => combobox.toggleDropdown()}
            {...rest}
          >
            <Group wrap="nowrap" gap="xs" justify={size === 'xl' ? 'space-between' : 'center'}>
              <Group gap="xs">
                {selectedOption?.Icon && <selectedOption.Icon size={40} />}

                {size === 'xl' ? (
                  <Heading order={5} c="currentColor">
                    {' '}
                    {selectedOption?.label || placeholder}
                  </Heading>
                ) : (
                  <BodyText c="currentColor">{selectedOption?.label || placeholder}</BodyText>
                )}

                {selectedOption?.additionalInfo && (
                  <Heading order={6} c="var(--mantine-color-neutrals-fills-medium)">
                    {selectedOption.additionalInfo}
                  </Heading>
                )}
              </Group>
              {rightIcon || <Icon.ArrowDropdown color="currentColor" size={10} />}
            </Group>
          </UnstyledButton>
        </Combobox.Target>
        <Combobox.Dropdown>
          {withSearch && (
            <SearchForTables
              placeholder={searchPlaceholder}
              value={search}
              setValue={(val) => setSearch(val)}
            />
          )}
          <Combobox.Options py="2n">{options}</Combobox.Options>
        </Combobox.Dropdown>
      </Combobox>
    </div>
  );
};
