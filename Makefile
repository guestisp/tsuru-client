# Copyright 2016 tsuru-client authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

# Python interpreter path
PYTHON := $(shell which python)


release:
	@if [ ! $(version) ]; then \
		echo "version parameter is required... use: make release version=<value>"; \
		exit 1; \
	fi

	@echo "Releasing tsuru $(version) version."

	@echo "Replacing version string."
	@sed -i "" "s/version = \".*\"/version = \"$(version)\"/g" tsuru/main.go

	@git add tsuru/main.go
	@git commit -m "bump to $(version)"

	@echo "Creating $(version) tag."
	@git tag $(version)

	@git push --tags
	@git push origin master

	@echo "$(version) released!"

doc-requirements:
	@pip install -r requirements.txt

docs-clean:
	@rm -rf ./docs/build

doc: docs-clean doc-requirements
	@tsuru_sphinx tsuru docs/ && cd docs && make html SPHINXOPTS="-N -W"

docs: doc

test:
	go test $$(go list ./... | grep -v /vendor/) -check.vv

install:
	go install $$(go list ./... | grep -v /vendor/)

build-all:
	./misc/build-all.sh

.PHONY: doc docs release manpage
