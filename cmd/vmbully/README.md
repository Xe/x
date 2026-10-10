# vmbully

`vmbully` runs JavaScript files against Proxmox VMs that are used one time, then deleted.

Each `vmbully script` run uses one VM from a pool. The pool is all VMs with the tag `vmbully`. After the script ends, `vmbully` deletes the VM and clones a new one for the next run.

## Requirements

- A Proxmox API token that can create, change, and delete VMs.
- A network route from your machine to the VM network. `vmbully` connects to port 22 of the VM.
- A VM template that has the QEMU guest agent and permits SSH password logins. `qemu_guest_agent.yaml` in this directory is a cloud-init vendor snippet that does both.

## Configuration

Set the connection flags before the subcommand name, or set them as environment variables.

| Flag                 | Environment variable | Default  |
| -------------------- | -------------------- | -------- |
| `--proxmox-host`     | `PROXMOX_HOST`       | none     |
| `--proxmox-token-id` | `PROXMOX_TOKEN_ID`   | none     |
| `--proxmox-secret`   | `PROXMOX_SECRET`     | none     |
| `--proxmox-node`     | `PROXMOX_NODE`       | `shachi` |

`PROXMOX_HOST` is the API URL, for example `https://shachi:8006/api2/json`.

Each subcommand flag also has an environment variable. Change the flag name to upper case and replace each `-` with `_`.

## Set up the pool

Do these steps one time.

1. Copy the vendor snippet to the Proxmox node.

   ```sh
   scp qemu_guest_agent.yaml root@shachi:/var/lib/vz/snippets/
   ```

   The `local` storage must have the `snippets` and `import` content types.

2. Create the template.

   ```sh
   vmbully template --id=9010 --ssh-keys-file=$HOME/.ssh/id_ed25519.pub \
     --ci-vendor=local:snippets/qemu_guest_agent.yaml
   ```

3. Optional: put the first VM in the pool. Without this step, the first `vmbully script` run clones its own VM.

   ```sh
   vmbully clone --template-id=9010
   ```

   The command prints the IP address and the password of the VM when the VM is ready.

## Run a script

```sh
vmbully script ./setup.js
```

`vmbully script` does these steps:

1. It selects at random a VM that is in operation, has the tag `vmbully`, and does not have the tag `tainted`. If there is none, it clones a new VM from the template with the tags `vmbully` and `tainted`.
2. It waits for the guest agent, then sets a random password for the user.
3. It connects to the VM with SSH.
4. It adds the tag `tainted` to the VM.
5. It runs the script.
6. It stops the VM and deletes it with its disks.
7. It clones a new VM from the template, tags it `vmbully`, and starts it.

Steps 6 and 7 also run when the script fails and when you press Ctrl-C.

If a step before step 4 fails, a VM from the pool stays in the pool. A VM that step 1 cloned is deleted, and step 7 does not run.

### Flags

| Flag             | Default   | Function                                     |
| ---------------- | --------- | -------------------------------------------- |
| `--tag`          | `vmbully` | The tag of the pool.                         |
| `--user`         | empty     | The SSH user. Empty is the cloud-init user.  |
| `--template-id`  | `9010`    | The template that the next VM is a clone of. |
| `--disk-storage` | `ssd`     | The storage for the disks of the next VM.    |
| `--disk-size`    | `32G`     | The size of the `scsi0` disk of the next VM. |

Put the flags before the script path.

### Exit status

| Status | Cause                                                                  |
| ------ | ---------------------------------------------------------------------- |
| `0`    | The script ended without an exception, and the cleanup was successful. |
| `1`    | The script threw an exception, or a Proxmox or SSH step failed.        |
| `2`    | The arguments are incorrect.                                           |

## Script functions

The script is one JavaScript file. `vmbully` evaluates it with QuickJS as a global script, not as a module. All functions are synchronous.

### `` $`command` ``

Runs `command` in the VM with the login shell of the user.

- The return value is the stdout of the command as a string. The string includes the last newline. Use `.trim()` to remove it.
- The stderr of the command goes to your terminal.
- A nonzero exit status throws an exception.
- Each `${value}` becomes one quoted shell word. An array becomes one word for each element.

```js
const kernel = $`uname -r`.trim();
const files = ["/etc/hostname", "/etc/os-release"];
console.log($`cat ${files}`);
```

Because `${value}` is always quoted, write shell syntax such as `|` and `>` in the fixed text of the command.

### `` host$`command` ``

Runs `command` with `sh` on your machine, not in the VM.

- The return value, the stderr, the exception, and the `${value}` quoting are the same as for `$`.
- The command starts in the directory where you started `vmbully`.

```js
const commit = host$`git rev-parse --short HEAD`.trim();
host$`make build`;
scp("./bin/app", "/tmp/");
$`/tmp/app --version`;
console.log(commit);
```

### `scp(source, dest)`

Copies the local file `source` to `dest` in the VM.

- `source` is a path on your machine. It must be a regular file.
- `dest` is a file path or a directory in the VM.
- The copy keeps the permission bits of the file.
- An error throws an exception.

```js
scp("./install.sh", "/tmp/");
$`sh /tmp/install.sh`;
```

### `console.log(...)` and `print(...)`

Write text to your terminal.

### Errors

An exception that the script does not catch stops the script. `vmbully` then logs the error and exits with status `1`.

Use `try` and `catch` when a command failure is permitted.

```js
try {
  $`systemctl is-active nginx`;
} catch (e) {
  console.log("nginx is not in operation");
}
```

## Example

```js
console.log($`uname -a`);

$`sudo apt-get install -y nginx`;
scp("./index.html", "/tmp/");
$`sudo mv /tmp/index.html /var/www/html/index.html`;

const page = $`curl -fsS http://localhost/`;
if (!page.includes("hello")) {
  throw new Error("the page does not have the expected text");
}
```

## Other commands

| Command                  | Function                                             |
| ------------------------ | ---------------------------------------------------- |
| `vmbully template`       | Creates a VM template from a cloud image.            |
| `vmbully clone`          | Clones a template into a new VM in the pool.         |
| `vmbully passwd <vmid>`  | Sets a new password in a VM and prints the password. |
| `vmbully help <command>` | Shows the flags of a command.                        |

## Limits

- Two `vmbully script` runs at the same time can use the same VM. The random selection makes this less probable, but there is no lock.
- `vmbully` does not check the SSH host key of the VM. Use it on a trusted network only.
- With no free VM in the pool, `vmbully script` clones one first. That run is slower, because the VM must complete its first boot.
- If the cleanup fails, a VM with the tag `tainted` can stay on the node. Delete it manually.
