# reviewbot

A small tool to feed code changes into an AI model using an OpenAI-compatible API and submit feedback on those changes to GitHub.

## Installation

Configure reviewbot to run [via GitHub Actions](./actions/). Then whenever you want a review, make a comment on GitHub containing the following text:

> /reviewbot

reviewbot will get started by GitHub Actions, churn through your code changes, and give (hopefully) helpful review feedback.

## Hacking

To hack at reviewbot, edit its code. This will allow you to add or remove functionality.

## Known limitations

- Some commits may be bigger than the context window of your AI model. Don't have reviewbot review big commits and you'll be fine.
- The feedback the model gives is more likely to be more useful if you're using a fairly beefy model.
- Does not work with closed source repos. Donate on [GitHub Sponsors](https://github.com/sponsors/Xe) to entice me to fix this as reviewbot Pro!
- Probably will crash if any data doesn't match what I saw in testing.
