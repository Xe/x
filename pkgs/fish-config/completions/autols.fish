set -l c complete -c autols

$c -f # Disable file completion

$c -s h -l help
set -l subcommands on off status toggle ignore

set -l cond "not __fish_seen_subcommand_from $subcommands"
$c -n "not __fish_seen_subcommand_from $subcommands" -a on
$c -n "not __fish_seen_subcommand_from $subcommands" -a off
$c -n "not __fish_seen_subcommand_from $subcommands" -a status
$c -n "not __fish_seen_subcommand_from $subcommands" -a toggle
$c -n "not __fish_seen_subcommand_from $subcommands" -a ignore

# FIX: why does the above arguments show up when `autols ignore |`
set -l ignore_subcommands add list ls remove rm
set -l cond "__fish_seen_subcommand_from ignore; and not __fish_seen_subcommand_from $ignore_subcommands"
$c -n "__fish_seen_subcommand_from ignore; and not __fish_seen_subcommand_from $ignore_subcommands" -a add
$c -n "__fish_seen_subcommand_from ignore; and not __fish_seen_subcommand_from $ignore_subcommands" -a "ls list"
# TODO: show the ignored list for `autols ignore remove`
$c -n "__fish_seen_subcommand_from ignore; and not __fish_seen_subcommand_from $ignore_subcommands" -a "rm remove"
