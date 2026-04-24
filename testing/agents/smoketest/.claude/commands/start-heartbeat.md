---
description: Register the heartbeat cron schedule
disable-model-invocation: true
---

Set up the smoketest agent schedule:

1. Use CronCreate to schedule `/heartbeat` to run every 10 minutes using the cron expression `0,10,20,30,40,50 * * * *`

After creating the schedule, confirm:
- The cron expression used
- When the next execution will be
- A reminder about the 7-day CronCreate expiry
