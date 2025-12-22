IMK
============
Simple file watcher similar to fswatch or inotify-wait.

Usage:
------
```bash
$ ./imk -h
Usage of ./imk:
  -c, --command string       command to execute when file is modified.
  -i, --immediate            run command immediately before watching for events.
  -o, --once                 run command once and exit on event.
  -r, --recurse              if a directory is supplied, add all its sub-directories as well.
  -u, --run string           command to execute if primary command succeeded.
  -d, --teardown string      teardown command to execute when -k timeout occurs (assumes -w). The PID is available in CMD_PID environment variable.
  -t, --threshold duration   number of seconds to skip after the last executed command (default: 0).
  -k, --timeout duration     timeout after which to kill the command subproces (default - do not kill).
  -v, --version              print version and exit.
```

To monitor all files and run a command on change, do the following:

```bash
$ imk -rc 'go build ./...' src/
:: [21:09:57] start monitoring: cmd[go build ./...] recurse files[src/]
:: [21:10:12] === src//main.rs (1) ===
    ...
:: [21:10:13] === exit code 0 ===
```

If any of the monitored files are modified, the command will be executed.
