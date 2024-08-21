---
title: "Why is hdrwtch in my logs?"
date: 2024-08-21
updated: 2024-08-21
slug: "/why-in-logs"
---

hdrwtch is showing up in your logs because a user is monitoring your website. This allows that user to receive notifications when the contents of a specific URL change. The user can also monitor the status of the website and receive notifications when the website goes down or comes back up.

If you want to block hdrwtch from monitoring your website, you can do so filtering out the user agent string `hdrwtch` in your server configuration. This will prevent hdrwtch from accessing your website and monitoring its contents, but it will also prevent the user from receiving notifications about your website.

hdrwtch is configured to check URLs every 15 minutes.

If you have any questions or concerns about hdrwtch, please [contact us](/docs/contact). We are happy to help you with any issues you may have.
