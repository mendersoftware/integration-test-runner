#!/bin/sh

pip install pytest-golden

py.test -s --tb=short -vv /testing/tests "$@"
