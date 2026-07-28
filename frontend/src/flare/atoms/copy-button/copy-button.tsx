import { CopyButton as MantineCopyButton } from '@mantine/core';
import { Tooltip } from '../tooltip/tooltip';
import { ActionIcon } from '../action-icon/action-icon';
import { Icon } from '../icons';

export interface CopyButtonProps {
  value: string;
  size?: number;
}
export const CopyButton = ({ value, size }: CopyButtonProps) => (
  <MantineCopyButton value={value} timeout={2000}>
    {({ copied, copy }) => (
      <Tooltip label={copied ? 'Copied' : 'Copy'}>
        <ActionIcon aria-label="Copy" onClick={copy}>
          {copied ? (
            <Icon.Check variant="gray" size={size || 16} />
          ) : (
            <Icon.Copy variant="gray" size={size || 16} />
          )}
        </ActionIcon>
      </Tooltip>
    )}
  </MantineCopyButton>
);
