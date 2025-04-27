# Playbooks

A playbook is a series of instructions that Alvis should perform on your behalf when running into issues. Playbooks are written in JSON and are executed by Alvis when the stated problem occurs.

For example, consider what you would do to restart a Fly app after the health check fails. restart the app.

```json
{
  "meta": {
    "service": "xe-pronouns",
    "condition": "health check failed"
  },

  "health_check_url": "https://pronouns.within.lgbt/.within/health",
  "details": "Run your own copy of health checks.\n\nIf your health check fails, restart the app.\nIf it succeeds, close the incident.\n\nWait for one minute afte restarting the app.\nRun the health check again after restarting the app.\n\nIf it fails again, escalate to the on-call engineer."
}
```
