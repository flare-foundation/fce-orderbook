---
description: Register the heartbeat cron schedule
disable-model-invocation: true
---

Set up the edge case agent schedule:

1. Use CronCreate to schedule `/heartbeat` to run every 10 minutes (offset +3) using the cron expression `3,13,23,33,43,53 * * * *`

After creating the schedule, confirm:
- The cron expression used
- When the next execution will be
- A reminder about the 7-day CronCreate expiry
