---
title: "Alerts"
date: 2024-08-21
updated: 2024-08-21
slug: "/alerts"
---

When the Last-Modified header of a URL changes, hdrwtch sends an alert to the user who is monitoring that URL. The alert contains the URL that was monitored, the previous Last-Modified header value, the new Last-Modified header value, and the time of the change.

Here is an example alert:

<div class="flex items-start gap-2.5 not-prose">
   <div class="flex flex-col w-full max-w-[320px] leading-1.5 p-4 border-gray-200 bg-gray-100 rounded-e-xl rounded-es-xl dark:bg-gray-700">
      <p class="text-sm font-normal py-2.5 text-gray-900 dark:text-white"><span class="font-medium">Changing route</span>:<br /><br />Last modified: Wed, 21 Aug 2024 18:18:15 GMT<br />Region: yow-dev<br />Status code: 200<br />Remark:</p>
   </div>
</div>

The alert is sent to the user via the messaging service that the user has configured in their account settings, but it defaults to Telegram if no other service is configured.

If there is an error making the request, the remark field will contain the error message.
