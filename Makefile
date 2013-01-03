PROJECT  := psm
# turns the git-describe output of v0.3-$NCOMMITS-$SHA1 into
# the more deb friendly 0.3.$NCOMMITS
VERSION  := $(shell git describe --long | sed 's/v\([0-9]*\)\.\([0-9]*\)-\([0-9]*\).*/\1.\2.\3/')

prefix   := /usr
bindir   := $(prefix)/bin
sharedir := $(prefix)/share
mandir   := $(sharedir)/man
man1dir  := $(mandir)/man1

all: build

build:
	go test
	go build

psm: *.go
	go build

clean:
	rm -rf psm build

install: psm
	install -m 4755 -o root psm $(DESTDIR)$(bindir)
	install -m 0644 psm.1 $(DESTDIR)$(man1dir)

deb: builddeb

builddeb:
	dch --newversion $(VERSION) --distribution unstable --force-distribution -b "Last Commit: $(shell git log -1 --pretty=format:'(%ai) %H %cn <%ce>')"
	dch --release  "new upstream"
	mkdir -p build
	git archive --prefix="$(PROJECT)-$(VERSION)/" HEAD | tar -xC build
	echo $(VERSION) >build/$(PROJECT)-$(VERSION)/version.txt
	(cd build/$(PROJECT)-$(VERSION) && debuild -us -uc -v$(VERSION))
	@echo "Package is at build/$(PROJECT)_$(VERSION)_all.deb"

version:

.PHONY: all build install deb builddeb version clean
