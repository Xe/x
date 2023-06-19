# `e621_reg_dropper`

This is a code snippet from the script kiddie that claimed to have
access to the database for e621. They claimed that this access would
let them dump a database of all e621 users.

After a month no such database has been released.

The Go program in this folder will create a `.reg` file that
automatically downloads and runs an arbitrary program that the
attacker specifies. It additionally tries to cloak itself by inserting
a bunch of garbage into the registry. The attacker-defined program
will run when the machine reboots, allowing a gap between infection
and activation.

Somehow, these generated `.reg` files are not detected by virus
scanners and a social engineering attack would be required to use this
as a stage in a longer attack.

This is overwhelmingly bad code though, I wouldn't let this pass in
code reviews.
