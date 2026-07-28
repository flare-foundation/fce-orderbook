import { Group, Stack, type NumberInputProps } from '@mantine/core';
import { Icon } from '../../atoms/icons';
import { TokenIcon, TokenIconName } from '../../atoms/token-icon/token-icon';
import { Tooltip } from '../../atoms/tooltip/tooltip';
import { BodyText } from '../../typography/body-text';
import { Heading } from '../../typography/heading';
import { AutoSizingNumberInput } from './auto-sizing-number-input';
import classes from './swap-values-input.module.css';

export type SwapValuesInputProps = {
  variant?: 'input' | 'default';
  mode: 'simple' | 'lots';
  token?: TokenIconName;
  amount?: string | number;
  infoTooltip?: string;
  max?: number;

  lots?: number | string;
  valueUsd?: number | string;
  onChange?: (val: number | string) => void;
  inputProps?: Omit<NumberInputProps, 'value' | 'onChange'>;
};

export const SwapValuesInput = ({
  mode,
  variant = 'input',
  token,
  amount,
  lots,
  max,
  valueUsd,
  infoTooltip,
  onChange,
}: SwapValuesInputProps) => {
  const value = mode === 'lots' ? lots : amount;
  const isLotsMode = mode === 'lots';
  const isEditable = variant === 'input';
  const subtextValue = isLotsMode ? `${amount} (${valueUsd})` : `${valueUsd}`;

  return (
    <Stack gap="0" className={classes.container}>
      <Group gap="xs" wrap="nowrap">
        {isEditable ? (
          <AutoSizingNumberInput value={Number(value) || 0} max={max} onChange={onChange} />
        ) : (
          <Heading order={1} className={classes.displayValue}>
            {value || '0'}
          </Heading>
        )}

        {isLotsMode && (
          <>
            <Heading order={5} className={classes.lotsLabel}>
              Lots
            </Heading>
            {infoTooltip && (
              <Tooltip label={infoTooltip}>
                <Icon.InfoCircle size={20} variant="default" />
              </Tooltip>
            )}
          </>
        )}
      </Group>

      <Group gap="3n">
        {isLotsMode && token && <TokenIcon name={token} size={24} />}
        <BodyText className={classes.valueText}>{subtextValue}</BodyText>
      </Group>
    </Stack>
  );
};
