all: clean_build package
clean_build: clean make-cred-helper make-config-edit

TRAVIS_BUILD_DIR ?= .
TRAVIS_TAG ?= no_tags
BIN_DIR = ${TRAVIS_BUILD_DIR}/bin
ARTIFACTS_DIR = ${TRAVIS_BUILD_DIR}/artifacts
GOARCH ?= amd64

clean:
	bash build/build-clean.sh ${BIN_DIR} ${ARTIFACTS_DIR}

make-cred-helper:
	bash build/build-cred-helper.sh ${BIN_DIR} ${GOARCH}

make-config-edit:
	bash build/build-config-edit.sh ${BIN_DIR} ${GOARCH}

package:
	bash build/package.sh ${BIN_DIR} ${ARTIFACTS_DIR}
