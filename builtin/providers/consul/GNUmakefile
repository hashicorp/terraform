# env TESTARGS='-test.parallel=1 -run TestAccDataConsulAgentSelf_basic' TF_LOG=debug make test
test::
	2>&1 env \
		make -C ../../.. testacc TEST=./builtin/providers/consul | tee test.log
