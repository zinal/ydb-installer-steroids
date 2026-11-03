.PHONY: help build runtime check

help:
	@printf '%s\n' \
		'make build    — собрать инсталлятор с плейбуками из ansible-collection/' \
		'make runtime  — собрать только встроенный runtime с этими плейбуками' \
		'make check    — проверить обвязку сборки, не запуская её'

build:
	bash scripts/build-installer.sh

runtime:
	bash scripts/build-installer.sh --runtime-only

check:
	bash scripts/build-installer.sh --self-test
