psm - simple, accurate memory reporting for Linux
=================================================

psm is based off the ideas and implementation of
[ps_mem.py](https://github.com/pixelb/scripts/commits/master/scripts/ps_mem.py).
It is implemented in go.

example output
--------------

    bpowers@python-worker-01:~$ sudo /usr/bin/time ./psm
      MB TOTAL      PRIV      SWAP	PROCESS (COUNT)
          0.01      0.00      0.10	/bin/sh (1)
          0.01      0.00      0.14	acpid (1)
          0.01      0.00      0.47	dhclient3 (1)
          0.05      0.03      0.14	atd (1)
          0.08      0.00      1.27	udevd (3)
          0.11      0.08      0.14	cron (1)
          0.11      0.00      1.23	whoopsie (1)
          0.13      0.09      0.48	memcached (1)
          0.13      0.02      0.93	getty (6)
          0.17      0.12      0.48	sshd (1)
          0.19      0.10      0.10	upstart-udev-bridge (1)
          0.19      0.10          	time (1)
          0.20      0.12      0.08	upstart-socket-bridge (1)
          0.35      0.32      0.16	dbus-daemon (1)
          0.35      0.30      0.47	ntpd (1)
          0.57      0.21      0.00	sshd: bpowers [priv] (1)
          0.60      0.55      0.33	rsyslogd (1)
          1.03      0.92      0.14	init (1)
          1.03      0.77      0.32	sudo (2)
          1.44      1.06      0.00	sshd: bpowers@pts/1 (1)
          3.03      3.03          	./psm (1)
          6.14      6.08          	-bash (1)
         11.81     11.70      6.68	node (1)
         26.26     25.76     43.99	[celeryd@notifications:MainProce (1)
         58.13     57.72     93.62	[celeryd@health:MainProcess] -ac (1)
        104.02    103.44      6.20	[celeryd@bulk:MainProcess] -acti (1)
        165.99    165.64    538.46	[celeryd@lookbacks:MainProcess]  (1)
        210.32    209.52          	[celeryd@stats:MainProcess] -act (1)

psm makes it easy to see who is resident in memory, and who is
swapped out.  The total column is the sum of the Pss of each mapping
in `/proc/$PID/smaps`.