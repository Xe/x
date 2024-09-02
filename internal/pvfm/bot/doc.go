/*
Package bot contains some generically useful bot rigging for Discord chatbots.

This package works by defining command handlers in a CommandSet, and then dispatching
based on message contents. If the bot's command prefix is `;`, then `;foo` activates
the command handler for command `foo`.

A CommandSet has a mutex baked into it for convenience of command implementation.
*/
package bot
