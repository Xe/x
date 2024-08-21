---
title: "hdrwtch"
date: 2024-08-21
updated: 2024-08-21
slug: "/"
---

hdrwtch is a tool that watches for changes in the `Last-Modified` header of a URL. You can use this to monitor the freshness of a web page, or to trigger an action when a page is updated.

## Setup

1. [Log in](/login) with Telegram.
2. Configure [a new probe](/probe).
3. Sit back and wait for updates.

[Alerts](/docs/alerts) will be sent to you via Telegram when a probe detects a change in the `Last-Modified` header. They will be aggregated into [reports](/docs/reports) that you can view at any time.

## Pricing

hdrwtch is free to use for up to 5 probes. For more than 5 probes, you will need to [upgrade to a paid plan](/docs/pricing).

## FAQ

Here are some frequently asked questions about hdrwtch and their answers.

### Why is hdrwtch in my logs?

A user configured a probe to watch a URL that you control. This is not a security risk. Please see ["Why is hdrwtch in my logs?"](/docs/why-in-logs) for more information.
