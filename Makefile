NAME=githublinter
VERSION=$(shell git describe --always --match v[0-9]* HEAD | cut -c2-)
BUILD_DIR=build
PACKAGE_DIR=$(BUILD_DIR)/$(NAME)-$(VERSION)
PACKAGE=$(BUILD_DIR)/$(NAME)-$(VERSION).deb

$(BUILD_DIR):
	@mkdir -p "$@"

.PHONY: githublinter
$(BUILD_DIR)/githublinter: $(BUILD_DIR)
	go build -o $@ ./cmd/githublinter

$(PACKAGE_DIR)/usr/bin/githublinter: $(BUILD_DIR)/githublinter
	@mkdir -p "$(dir $@)"
	cp -p "$<" "$@"

.PHONY: deb
deb: $(PACKAGE)

$(PACKAGE): \
	$(PACKAGE_DIR)/usr/bin/githublinter \
	$(PACKAGE_DIR)/DEBIAN/conffile \
	$(PACKAGE_DIR)/DEBIAN/control
	fakeroot dpkg-deb --build "$(PACKAGE_DIR)"

$(PACKAGE_DIR)/DEBIAN/%: debian/%
	@mkdir -p "$(dir $@)"
	cp -p "$<" "$@"

$(PACKAGE_DIR)/DEBIAN/control: debian/control
	@mkdir -p "$(dir $@)"
	(cat $< && printf 'Version: %s\n' "${VERSION}") > "$@"

.PHONY: release
release: clean $(PACKAGE)
	@mkdir -p "$(dir $@)"
	@[ -z "$(shell git status --porcelain)" ] || (echo "Cannot release with unclean working directory"; false)
	hub release create --attach="$<" "$(VERSION)"

.PHONY: clean
clean:
	rm -rf "$(BUILD_DIR)"
