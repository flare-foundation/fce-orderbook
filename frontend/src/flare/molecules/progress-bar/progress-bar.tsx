import { Flex, Progress, ProgressProps } from '@mantine/core';
import { BodyText } from '../../typography/body-text';

export interface ProgressBarProps extends Omit<ProgressProps, 'value'> {
  label?: string;
  currentStep: number;
  totalSteps: number;
  hideSteps?: boolean;
}
export const ProgressBar = ({
  label,
  hideSteps,
  currentStep,
  totalSteps,
  ...rest
}: ProgressBarProps) => {
  return (
    <Flex gap="3n" h="fit-content" w="100%" direction="column">
      {' '}
      {!hideSteps ? (
        <BodyText c="var(--mantine-color-neutrals-fills-black)" pl="2n">
          Step {currentStep}/{totalSteps}
          {label ? ':' : ''}
        </BodyText>
      ) : (
        <></>
      )}
      {label ? <BodyText c="var(--mantine-color-neutrals-fills-medium)">{label}</BodyText> : <></>}
      <Progress
        w="100%"
        styles={{ section: { borderRadius: '32px' } }}
        bg="var(--mantine-color-neutrals-fills-light)"
        radius="32px"
        color="var(--mantine-color-token-pairs-charting-lavender)"
        value={(100 * currentStep) / totalSteps}
        h="md"
        {...rest}
      />
    </Flex>
  );
};
