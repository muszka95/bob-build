#!/usr/bin/env bash
source .travis/utils.sh
STATUS_CODE=0 # reset

####################
fold_start 'Relative path tests'
    bash tests/relative_path_tests.sh
fold_end $?
####################

####################
fold_start 'Build tests'
    bash tests/build_tests.sh
    build_result=$?
fold_end ${build_result}
####################

####################
fold_start 'Build example tests'
    bash .travis/build_example_proj.sh
fold_end $?
####################

# Tests of go code (python version doesn't matter)
if [ ${DO_GO_TESTS} -eq 1 ] ; then
    ####################
    fold_start 'Go tests'
        bash .travis/run_go_tests.sh
    fold_end $?
    ####################
fi

# Tests of python code (go version doesn't matter)
if [ ${DO_PYTHON_TESTS} -eq 1 ] ; then
    ####################
    fold_start 'config_system regression tests'
        config_system/tests/run_tests.py
    fold_end $?
    ####################

    ####################
    fold_start 'Mconfigfmt tests'
        config_system/tests/run_tests_formatter.py
    fold_end $?
    ####################

    ####################
    fold_start 'config_system pytest'
        pytest config_system
    fold_end $?
    ####################

    ####################
    fold_start 'scripts pytest'
        pytest scripts/env_hash.py
    fold_end $?
    ####################
fi

####################
# This test is issued only if build test passed previously.
# We do this last as this test changes the checkout
if [[ ${build_result} == 0 ]];then
    fold_start 'Bootstrap version test'
        bash .travis/run_bootstrap_test.sh
    fold_end $?
else
    result_skip 'Bootstrap version test'
fi

####################

exit $STATUS_CODE
