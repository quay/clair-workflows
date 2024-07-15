SRCS = $(wildcard */workflow.dhall)
YMLS = $(patsubst %/workflow.dhall,.github/workflows/%.yml,$(SRCS))
ALL_DHALL = $(wildcard *.dhall */*.dhall)

to_generate = actions-versions
GEN_DEPS = $(foreach f,$(to_generate),lib/$f_generated.dhall)

.PHONY: all clean check fmt

all: $(YMLS)

clean:
	rm -f $(YMLS) $(GEN_DEPS)

check: $(ALL_DHALL)
	dhall lint --check $^
	dhall format --check $^

fmt: $(ALL_DHALL)
	dhall lint $^
	dhall format $^
	dhall freeze $(filter %/workflow.dhall,$^)

.github/workflows/%.yml: %/workflow.dhall %/* *.dhall lib/*.dhall $(GEN_DEPS)
	dhall-to-yaml --file $< |\
		sed '/^[[:space:]]\+$$/s/[[:space:]]//g' >$@

lib/actions-versions_generated.dhall: lib/actions-versions.dhall Makefile
	dhall-to-json < $< |\
		jq 'map(capture("^(?<value>[^/]+/(?<key>[^@]+)@.+)$$")) | from_entries' |\
		json-to-dhall >$@
	dhall format $@
