+++
title = "Environment Variables"
weight = 40
+++

## GAMEFILE_VERBOSE

Set to "1" or "true" to turn on verbose mode (like running with -v)

## GAMEFILE_DEBUG 

Set to "1" or "true" to turn on debug mode (like running with -debug)

## GAMEFILE_CACHE

Sets the directory where mage will store binaries compiled from magefiles
(default is $HOME/.magefile)

## GAMEFILE_GOCMD

Sets the binary that mage will use to compile with (default is "go").

## GAMEFILE_IGNOREDEFAULT

If set to "1" or "true", tells the compiled magefile to ignore the default
target and print the list of targets when you run `mage`.

## GAMEFILE_HASHFAST

If set to "1" or "true", tells mage to use a quick hash of magefiles to
determine whether or not the magefile binary needs to be rebuilt. This results
in faster run times (especially on Windows), but means that mage will fail to
rebuild if a dependency has changed. To force a rebuild when you know or suspect
a dependency has changed, run mage with the -f flag.
