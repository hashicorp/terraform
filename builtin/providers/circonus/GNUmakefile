# env TESTARGS='-test.parallel=1 -run TestAccCirconusCheckBundle' TF_LOG=debug make test
test::
	2>&1 make -C ../../.. testacc TEST=./builtin/providers/circonus | tee test.log
