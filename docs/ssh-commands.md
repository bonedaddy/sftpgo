# SSH commands

Some SSH commands are implemented directly inside SFTPGo, while for others we use system commands that need to be installed and in your system's `PATH`.

For system commands we have no direct control on file creation/deletion and so there are some limitations:

- we cannot allow them if the target directory contains virtual folders or file extensions filters
- system commands work only on local filyestem
- we cannot avoid to leak real filesystem paths
- quota check is suboptimal

 If quota is enabled and SFTPGO receives a system command, the used size and number of files are checked at the command start and not while new files are created/deleted. While the command is running the number of files is not checked, the remaining size is calculated as the difference between the max allowed quota and the used one, and it is checked against the bytes transferred via SSH. The command is aborted if it uploads more bytes than the remaining allowed size calculated at the command start. Anyway, we only see the bytes that the remote command sends to the local one via SSH. These bytes contain both protocol commands and files, and so the size of the files is different from the size trasferred via SSH: for example, a command can send compressed files, or a protocol command (few bytes) could delete a big file. To mitigate these issues, quotas are recalculated at the command end with a full scan of the directory specified for the system command. This could be heavy for big directories. If you need system commands and quotas you could consider disabling quota restrictions and periodically update quota usage yourself using the REST API.

 For these reasons we should limit system commands usage as much as possibile, we currently support the following system commands:

- `git-receive-pack`, `git-upload-pack`, `git-upload-archive`. These commands enable support for Git repositories over SSH. They need to be installed and in your system's `PATH`.
- `rsync`. The `rsync` command needs to be installed and in your system's `PATH`. We cannot avoid that rsync creates symlinks, so if the user has the permission to create symlinks, we add the option `--safe-links` to the received rsync command if it is not already set. This should prevent creating symlinks that point outside the home dir. If the user cannot create symlinks, we add the option `--munge-links` if it is not already set. This should make symlinks unusable (but manually recoverable).

SFTPGo support the following built-in SSH commands:

- `scp`, SFTPGo implements the SCP protocol so we can support it for cloud filesystems too and we can avoid the other system commands limitations. SCP between two remote hosts is supported using the `-3` scp option.
- `md5sum`, `sha1sum`, `sha256sum`, `sha384sum`, `sha512sum`. Useful to check message digests for uploaded files.
- `cd`, `pwd`. Some SFTP clients do not support the SFTP SSH_FXP_REALPATH packet type, so they use `cd` and `pwd` SSH commands to get the initial directory. Currently `cd` does nothing and `pwd` always returns the `/` path.
- `sftpgo-copy`. This is a built-in copy implementation. It allows server side copy for files and directories. The first argument is the source file/directory and the second one is the destination file/directory, for example `sftpgo-copy <src> <dst>`. The command will fail if the destination directory exists. Copy for directories spanning virtual folders is not supported. Only local filesystem is supported: recursive copy for Cloud Storage filesystems requires a new request for every file in any case, so a server side copy is not possibile. Please be aware that only the `list` permission for the source path and the `upload` and `create_dirs` (for directories) permissions for the destination path are checked.
- `sftpgo-remove`. This is a built-in remove implementation. It allows to remove single files and to recursively remove directories. The first argument is the file/directory to remove, for example `sftpgo-remove <dst>`. Only local filesystem is supported: recursive remove for Cloud Storage filesystems requires a new request for every file in any case, so a server side remove is not possibile.

The following SSH commands are enabled by default:

- `md5sum`
- `sha1sum`
- `cd`
- `pwd`
- `scp`
