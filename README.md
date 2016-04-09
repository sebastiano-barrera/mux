# mux

`mux` is a tiny UNIX utility that asynchronously reads from multiple
files.

Given a simple format string and a list of files, it will "listen to"
all the files at the same time, waiting for a new line to be
available. As soon as it is, the output is immediately updated.

Combining `mux` with named pipes (particularly through the `<()`
syntax available in many shells like bash), it's possible to achieve a
lot of useful behaviours (for an example, see _Example of usage_).

## Usage

```sh
mux <format string> [<file0> ... <fileN>]
```

The format string can contain format specifiers which look like `%0`,
`%1`, `%2` ..., each referring to the most recent line read from the 1st,
2nd, 3rd, ... file respectively.

## Example of usage

My original motivation for writing mux was feeding a `dzen2` bar from
multiple sources updating at different rates. Ideally, I wanted to use
just the shell and standard "off the shelf" coreutils, but I couldn't
find a way to combine them in order to get the behaviour I wanted.

Supposing that: information about the state of the window manager
comes from stdin, but a new line is received only when the state
changes; we want to show the current date, always updating
independently each second; we want to know the status of the battery,
although it's enough to update it each minute.  Note that updates from
stdin are _pushed_ to our script, while battery and date information
are _pulled_ (requested explicitly) at regular intervals (different
intervals for each source).  With mux, it's very easy to get to the
result:

```sh
function date_src {
    while true ; do
		date
		sleep 1
    done
}

function battery_src {
	battery=/org/freedesktop/UPower/devices/battery_BAT1
	while true ; do
		batstatus $battery  # a custom script
		sleep 60
	done
}

mux '%0 :: bat %2 | %1' /dev/stdin <(date_src) <(battery_src) | dzen2 -dock -ta l
```
