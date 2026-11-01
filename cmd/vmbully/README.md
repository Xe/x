# vmbully

`vmbully` runs JavaScript files in disposable Proxmox LXC containers.
Each script run creates a new unprivileged container from an archive, runs the script as root over SSH, then deletes the container.
There is no pool and no replacement container after a run.

## Requirements

- A Proxmox API token with permission to allocate, start, inspect, and delete containers and allocate storage.
- A downloaded Proxmox container archive on the target node.
- An archive with an SSH server and `scp` installed. Root must permit SSH public key logins.
- DHCP on the container network bridge.
- A route from your machine to port 22 on the container network.

Containers use the host kernel. Scripts that need their own kernel require a VM.

## Configuration

Set connection flags before the subcommand, or use environment variables.

| Flag                 | Environment variable | Default  |
| -------------------- | -------------------- | -------- |
| `--proxmox-host`     | `PROXMOX_HOST`       | none     |
| `--proxmox-token-id` | `PROXMOX_TOKEN_ID`   | none     |
| `--proxmox-secret`   | `PROXMOX_SECRET`     | none     |
| `--proxmox-node`     | `PROXMOX_NODE`       | `shachi` |

`PROXMOX_HOST` is the API URL, for example `https://shachi:8006/api2/json`.
Subcommand flags also have environment variables: use upper case and replace each `-` with `_`.

## Choose the base image

1. In Proxmox, select storage with the `vztmpl` content type, such as `local`.
2. Open **CT Templates**.
3. Download the Ubuntu template you want.
4. Use its volume ID with `--os-template`.

For example, if the downloaded archive has this name:

```sh
vmbully script \
  --os-template=local:vztmpl/ubuntu-24.04-standard_24.04-2_amd64.tar.zst \
  ./setup.js
```

Use the exact archive name from your storage. `--os-template` has no default.
The archive is a container filesystem, not a VM cloud image or a container ID.

You can also use an environment variable:

```sh
export OS_TEMPLATE=local:vztmpl/ubuntu-24.04-standard_24.04-2_amd64.tar.zst
vmbully script ./setup.js
```

## Run a script

Each `vmbully script` run does these steps:

1. Generate a temporary SSH key.
2. Allocate a new container ID.
3. Create an unprivileged container from `--os-template` with DHCP and the temporary public key.
4. Start the container and wait for its IPv4 address and SSH server.
5. Run the JavaScript file as root.
6. Stop the container and delete its disks.

Cleanup also runs after script errors, SSH setup errors, and Ctrl-C.
Completed Proxmox tasks with warnings continue. vmbully logs their warning status and warning messages.
After an accepted create request, vmbully waits for the task to finish before cleanup.
If the create task status is unknown, vmbully reports the container ID for manual inspection.
If an API or cleanup failure prevents deletion, the error reports the container ID.

### Flags

Put these flags before the script path.

| Flag             | Default  | Function                                          |
| ---------------- | -------- | ------------------------------------------------- |
| `--os-template`  | required | Volume ID of the downloaded container archive.    |
| `--disk-storage` | `ssd`    | Storage for the root filesystem.                  |
| `--disk-size`    | `32`     | Root filesystem size in GiB (integer).            |
| `--bridge`       | `vmbr0`  | Bridge for DHCP networking.                       |
| `--memory`       | `2048`   | Memory in MiB.                                    |
| `--cores`        | `2`      | CPU cores.                                        |
| `--nesting`      | `true`   | Nesting for compatibility with container systemd. |

### Exit status

| Status | Cause                                                       |
| ------ | ----------------------------------------------------------- |
| `0`    | The script and cleanup succeeded.                           |
| `1`    | The script, configuration, Proxmox API, or SSH step failed. |
| `2`    | The command arguments are incorrect.                        |

## Script functions

The script is one JavaScript file. `vmbully` evaluates it with QuickJS as a global script, not as a module. All functions are synchronous.

### `` $`command` ``

Runs `command` in the container with the login shell of root.

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

Runs `command` with `sh` on your machine, not in the container.

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

Copies the local file `source` to `dest` in the container.

- `source` is a path on your machine. It must be a regular file.
- `dest` is a file path or a directory in the container.
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

$`apt-get install -y nginx`;
scp("./index.html", "/tmp/");
$`mv /tmp/index.html /var/www/html/index.html`;

const page = $`curl -fsS http://localhost/`;
if (!page.includes("hello")) {
  throw new Error("the page does not have the expected text");
}
```

## Create a container for manual use

```sh
vmbully create --os-template="$OS_TEMPLATE" \
  --ssh-keys-file="$HOME/.ssh/id_ed25519.pub"
```

`create` uses the same resource flags as `script` and requires `--ssh-keys-file`.
It prints the container ID and leaves the container in operation.
Find the DHCP address in Proxmox, then connect as root with your SSH key.
Delete the container manually after use.

## Migration from VMs

The `template`, `clone`, and `passwd` commands no longer exist.
Use `create` for manual containers and `script` for disposable containers.
The `--template-id`, `--tag`, `--user`, and cloud-init flags no longer exist.
`--disk-size` now takes an integer in GiB, for example `32` instead of `32G`.
Existing VMs and templates stay on Proxmox. vmbully does not inspect or delete them.

## Limits

- The API connection skips TLS certificate verification, and SSH skips host key verification. Use vmbully on a trusted network.
- The temporary SSH private key stays in memory and disappears after the process exits.
- A forced process kill can leave a container on the node. Delete it manually.
- If cleanup fails, delete the reported container manually.
