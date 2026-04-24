---
description: Register the heartbeat cron schedule
disable-model-invocation: true
---

Set up the chaos agent schedule:

1. Use CronCreate to schedule `/heartbeat` to run every 10 minutes (offset +7) using the cron expression `7,17,27,37,47,57 * * * *`

After creating the schedule, confirm:
- The cron expression used
- When the next execution will be
- A reminder about the 7-day CronCreate expiry
