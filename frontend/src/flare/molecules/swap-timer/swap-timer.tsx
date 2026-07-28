import { Group, Stack } from '@mantine/core';
import { Icon } from '../../atoms/icons';
import { Loader } from '../../atoms/loader/loader';
import { BodyText } from '../../typography/body-text';
import { Heading } from '../../typography/heading';

export interface SwapTimerProps {
  timeStr: string;
  text?: string;
  secondaryTimer?: {
    text: string;
    status: 'loading' | 'success';
  };
}
export const SwapTimer = ({ timeStr, text, secondaryTimer }: SwapTimerProps) => {
  return (
    // vendored tweak: gap "xs" → "md" — the timer and its secondary line sat
    // nearly touching in our layouts.
    <Stack gap="md" opacity={0.99}>
      <Group gap="sm">
        <Loader size={24} />

        <Heading order={6}>{timeStr}</Heading>
      </Group>
      {text && (
        <BodyText lts="0.04em" type="lBody">
          {text}
        </BodyText>
      )}
      {secondaryTimer && (
        <Group gap="8px">
          {secondaryTimer.status === 'loading' ? (
            <Loader size={12} />
          ) : (
            <Icon.SuccessCircle size={12} variant="gray" />
          )}
          <BodyText
            type="note"
            lts="0.03em"
            fw={700}
            c="var(--mantine-color-neutrals-fills-medium)"
          >
            {secondaryTimer.text}
          </BodyText>
        </Group>
      )}
    </Stack>
  );
};
