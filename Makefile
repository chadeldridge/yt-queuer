# Makefile

#ROOT_DIR:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
BIN_DIR=bin
BUILD_DIR=build
WOLAPP_NAME=wol
YTAPP_NAME=ytqueuer
WOL_APP=
YT_APP=
GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)
ARCHIVE_DIR=${BUILD_DIR}/${YTAPP_NAME}

test:
	$(call get_arch)
	@echo "BIN_DIR: ${BIN_DIR}"
	@echo "BUILD_DIR: ${BUILD_DIR}"
	@echo "WOLAPP_NAME: ${WOLAPP_NAME}"
	@echo "WOL_APP: ${WOL_APP}"
	@echo "YTAPP_NAME: ${YTAPP_NAME}"
	@echo "YT_APP: ${YT_APP}"
.PHONY: test

tidy:
	@go mod tidy
.PHONY: tidy

build: tidy npm-build wol-build yt-build package
	@echo "Done.\n"
.PHONY: build

build-arm64: tidy npm-build wol-setup-arm64 wol-build yt-setup-arm64 yt-build package
	@echo "Done.\n"
.PHONY: build-arm64

run: tidy npm-build yt-build wol-build copy
	@cp -r certs ${ARCHIVE_DIR}/
	@cp -r db ${ARCHIVE_DIR}/
	@cd ${ARCHIVE_DIR} && ./${YTAPP_NAME} start
.PHONY: run

stop: build
	@bin/${YTAPP_NAME} stop
.PHONY: stop

wol-build:
	$(eval WOL_APP=$(shell ./tools/builder ${WOLAPP_NAME} ${GOOS} ${GOARCH}))
	@echo "WOL_APP: ${WOL_APP}"
.PHONY: build-wol

wol-build-arm64:
	$(eval GOARCH=arm64)
	$(eval WOL_APP=$(shell ./tools/builder ${WOLAPP_NAME} ${GOOS} ${GOARCH}))
	@echo "WOL_APP: ${WOL_APP}"
.PHONY: wol-build-arm64

npm-build:
	@echo "Building frontend..."
	@npm run -s build
.PHONY: npm-build

yt-build:
	$(eval YT_APP=$(shell ./tools/builder ${YTAPP_NAME} ${GOOS} ${GOARCH}))
	@echo "YT_APP: ${YT_APP}"
.PHONY: yt-build

yt-build-arm64:
	$(eval GOARCH=arm64)
	$(eval YT_APP=$(shell ./tools/builder ${YTAPP_NAME} ${GOOS} ${GOARCH}))
	@echo "YT_APP: ${YT_APP}"
.PHONY: yt-build-arm64

copy:
	@rm -rf ${ARCHIVE_DIR}
	$(eval ARCHIVE_DIR=$(shell ./tools/packager -c ${YTAPP_NAME} ${GOOS} ${GOARCH} \
		-f 'sys/*' \
		-f LICENSE \
		-f ${WOL_APP}:wol \
		-f ${YT_APP}:ytqueuer \
		-f public/))
.PHONY: copy

package:
	@rm -rf ${ARCHIVE_DIR}
	$(eval ARCHIVE_DIR=$(shell ./tools/packager ${YTAPP_NAME} ${GOOS} ${GOARCH} \
		-f 'sys/*' \
		-f LICENSE \
		-f ${WOL_APP}:wol \
		-f ${YT_APP}:ytqueuer \
		-f public/))
.PHONY: package

package-public:
	@if [ ! -d ${BUILD_DIR}/public ]; then \
		mkdir -p ${BUILD_DIR}/public/css && \
		mkdir ${BUILD_DIR}/public/js; \
		fi || exit 1
	@cp -r ${ROOT_DIR}/public/*.html ${BUILD_DIR}/public/
	@cp -r ${ROOT_DIR}/public/css/*.min.css ${BUILD_DIR}/public/css/
	@cp -r ${ROOT_DIR}/public/js/*.js ${BUILD_DIR}/public/js/
.PHONY: package-public

git-tags:
	@git tag -n
.PHONY: git-tags
