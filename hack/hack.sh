#!/usr/bin/env bash

# MIT License

# Copyright (c) 2021 Luca Sepe

# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:

# The above copyright notice and this permission notice shall be included in all
# copies or substantial portions of the Software.

# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
# SOFTWARE.

set -o errexit
set -o nounset
set -o pipefail

CODE_GEN=./vendor/k8s.io/code-generator/generate-groups.sh
chmod +x ${CODE_GEN}

# Replace this with your module name
MODULE_ROOT="github.com/lucasepe/expression-resolver"

${CODE_GEN} "all" \
  ${MODULE_ROOT}/pkg/generated \
  ${MODULE_ROOT}/pkg/apis \
  expression:v1alpha1 \
  --go-header-file ./hack/boilerplate.go.txt

# Hack to move the generated code in the right place
if [ -d "$MODULE_ROOT" ]; then
  cp -r ${MODULE_ROOT}/pkg/* ./pkg
  rm -rf "github.com"
fi
