[![Build](https://github.com/snobb/go-imk/actions/workflows/go.yml/badge.svg)](https://github.com/snobb/go-imk/actions/workflows/go.yml)


IMK
============
Simple file watcher similar to fswatch or inotify-wait.

Building:
---------

```bash
make
cp -f bin/imk /usr/local/bin/
```


Usage:
------
```plain
$ imk -h

Usage of imk:
  -c, --command string     primary command to execute when a file or a folder is modified.
  -i, --immediate          run commands immediately before watching for events.
  -n, --once               run primary command once and exit on event.
  -o, --output string      send the stdout of secondary command to a file.
  -r, --recurse            if a directory is supplied, add all its sub-directories as well.
  -u, --run string         secondary command to execute if primary command succeeded - runs in background.
  -k, --timeout duration   timeout after which to kill the command subprocess (default - do not kill).
  -v, --version            print version and exit. [main.14.da7d12e]

It is required to specify either primary or secondary command (or both).

The secondary command will run in the background and will be restarted immediately after the primary command is executed the next time.

Examples:
  imk -rc 'go build ./...' src/
  imk -rc 'go build ./...' src/ -k 5m
  imk -ric 'go build ./...' -u 'go run ./...' src/

```

To monitor all files and run a command on change, do the following:

```plain
$ imk -ric 'make dist' -u 'node --enable-source-maps dist/app.js' ./src/
:: 17:10:16 === watching files and folders: [./src/ ./src/ src/linestream src/log src/payment src/tcpserver src/test src/test/mocksocket] ===
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

If any of the monitored files are modified, the build command (-c flag) will be executed and if it's successful, the run command (-u) will be run (if it's running - it will be killed and restarted).
