[![Build](https://github.com/snobb/go-imk/actions/workflows/go.yml/badge.svg)](https://github.com/snobb/go-imk/actions/workflows/go.yml)

# IMK

Simple file watcher similar to fswatch or inotify-wait.

## Building:

```bash
make
sudo cp -f bin/imk /usr/local/bin/
```

## Usage:

```plain
$ ./bin/imk
Usage of ./bin/imk:
  -c, --command stringArray            command to execute on change (can be specified multiple times)
  -i, --immediate                      run commands immediately before watching for events.
  -n, --once                           run primary command once and exit on event.
  -o, --output string                  send the stdout of secondary command to a file with given prefix.
      --rate-limit-interval duration   Limit execution to one command per the given <interval>. (default 1.5s)
  -r, --recurse                        if a directory is supplied, add all its sub-directories as well.
  -k, --timeout duration               timeout after which to kill the command subprocess (default - do not kill).
  -v, --version                        print version and exit. [v1.0.0-2-ge225-dev]
  -w, --wrapshell                      wrap command with shell (default - false).

If multiple commands are specified, the first runs in the foreground as the primary command.
Subsequent commands run in the background and may be long-running.

Examples:
  imk -rc 'go build ./...' src/
  imk -rc 'go build ./...' src/ -k 5m
  imk -ric 'go build ./...' -c 'go run ./...' src/
```

To monitor all files and run a command on change, do the following:

```plain
$ imk -ric 'make dist' -c 'node --enable-source-maps dist/app.js' ./src/
:: 17:10:16 === watching files and folders: [./src/ ./src/main src/test] ===
rm -rf dist
./node_modules/.bin/tsc -p tsconfig-build.json
:: 17:10:18 === exit code 0 ===
listening on: 8888
:: 17:10:20 === CREATE :: src/payment/4913 ===
rm -rf dist
./node_modules/.bin/tsc -p tsconfig-build.json
:: 17:10:22 === exit code 0 ===
:: 17:10:22 === process killed by signal ===
listening on: 8888
```

If any of the monitored files are modified, the primary command (first -c flag)
will be executed and if it's successful, the rest of the commands will be run
in the background (other -c flags).
