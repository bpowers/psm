psm - simple, accurate memory reporting for Linux
=================================================

`psm` makes it easy to see who is resident in memory, and who is
significantly swapped out.

`psm` is based off the ideas and implementation of
[ps_mem.py](https://github.com/pixelb/scripts/commits/master/scripts/ps_mem.py).
It requires root privileges to run.  It is implemented in go, and
since the executable is a binary it can be made setuid root so that
unprivileged users can get a quick overview of the current memory
situation.

installation
------------

If you're familiar with go and have the go toolchain installed,
installation is as easy as:

    go get github.com/bpowers/psm
    sudo `which psm`

The ``sudo `which psm` `` can get a bit tiring.  If you're on
Ubuntu, there is a PPA which install psm as setuid root:

    sudo apt-get install python-software-properties # for apt-add-repository
    sudo add-apt-repository ppa:bobbypowers/psm
    sudo apt-get update
    sudo apt-get install psm

example output
--------------

    bpowers@python-worker-01:~$ psm -filter=celery
        MB RAM    SHARED   SWAPPED	PROCESS (COUNT)
          60.6       1.1     134.2	[celeryd@notifications:MainProcess] (1)
          62.6       1.1          	[celeryd@health:MainProcess] (1)
         113.7       1.2          	[celeryd@uploads:MainProcess] (1)
         155.1       1.1          	[celeryd@triggers:MainProcess] (1)
         176.7       1.2          	[celeryd@updates:MainProcess] (1)
         502.9       1.2          	[celeryd@lookbacks:MainProcess] (1)
         623.8       1.2      28.5	[celeryd@stats:MainProcess] (1)
         671.3       1.2          	[celeryd@default:MainProcess] (1)
    #   2366.7               164.7	TOTAL USED BY PROCESSES

The `MB RAM` column is the sum of the Pss value of each mapping in
`/proc/$PID/smaps` for each process.

TODO
----

- port to the BSDs and OS X
- FreeBSD has a Linux-compatable procfs impelmentation, which would
  be trivial to use (and, indeed, ps_mem.py uses it).
- OS X looks... fun.  MacFUSE provides a lot of the info we need, but
  I don't want to depend on having that installed and manually having
  their procfs mounted.  There are Mach functions we could use, but
  I'm having trouble figuring out how to correctly pass data between
  go and C.  Specifically: https://gist.github.com/4463209 - 'patches
  welcome'.
- ps_mem.py records the md5sum of each process's smaps entry to make
  sure that we're not double-counting.  Its probably worth doing.

license
-------

psm is offered under the MIT license, see LICENSE for details.
