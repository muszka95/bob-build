#!/usr/bin/env python

# Copyright 2018 Arm Limited.
# SPDX-License-Identifier: Apache-2.0
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import argparse
import filecmp
import logging
import os
import re
import shutil
import sys

RE_INCLUDE = re.compile(r'^\s*#include ["<](.*)[">]')
cached_deps = dict()

def get_include_statements(fname):
    # This reads kernel sources which could contain non-ascii
    # characters, so force utf-8.
    # This should only affect comments.
    with open(fname, "rb") as fp:
        content = fp.read().decode('utf-8')
        lines = content.split('\n')

    ret = []
    for line in lines:
        m = RE_INCLUDE.match(line)
        if m:
            ret += [m.groups()[0]]
    return ret

def search_for_include(include, search_path):
    for d in search_path:
        test_path = os.path.join(d, include)
        if os.path.isfile(test_path):
            return test_path
    return None

def get_includes(fname, search_path, visited, extra_includes=[]):
    global cached_deps

    if fname in visited:
        return []

    if fname in cached_deps:
        return cached_deps[fname]

    ret = []
    visited.append(fname)
    include_statements = get_include_statements(fname) + extra_includes
    for include in include_statements:
        path = search_for_include(include, [os.path.dirname(fname)] + search_path)
        if path:
            ret += [path]
            ret += get_includes(path, search_path, visited)

    ret = sorted(set(ret))

    cached_deps[fname] = ret
    return ret

def write_depfile(depfile, target_name, deps):
    with open(depfile, "wt") as fp:
        fp.write("%s: \\\n    " % target_name)
        fp.write(" \\\n    ".join(deps) + "\n")

def parse_args():
    ap = argparse.ArgumentParser()
    ap.add_argument("--depfile", "-d", metavar="DEPFILE", required=True)
    ap.add_argument("--include", "-i", metavar="FILE", action="append", default=[])
    ap.add_argument("--target-dir", "-t", metavar="DIR", required=True)
    ap.add_argument("--target-name", "-n", metavar="NAME", required=True)
    ap.add_argument("--include-dir", "-I", metavar="INCLUDE_DIR", action="append", default=[])
    ap.add_argument("source", nargs="+")
    return ap.parse_args()

def copy_if_newer(src, dest):
    try:
        os.makedirs(os.path.dirname(dest))
    except OSError:
        pass
    if not os.path.isfile(src):
        logging.error("%s does not exist (cwd=%s)", src, os.getcwd())
        sys.exit(1)

    # If source and dest both exist and are the same, skip the copy.
    if os.path.isfile(dest) and filecmp.cmp(src, dest):
        return

    shutil.copy(src, dest)

def copy_with_deps(src_rel, dest, search_path, includes):
    deps = []

    if os.path.isabs(src_rel):
        logging.error("%s is not a relative path", src_rel)
        sys.exit(1)

    ext = os.path.splitext(src_rel)[1]
    if ext in {".c", ".cpp", ".cxx"}:
        deps = get_includes(os.path.abspath(src_rel), search_path, [], extra_includes=includes)

    copy_if_newer(src_rel, dest)

    return deps

def main(args):
    logging.basicConfig(format='%(levelname)s: %(message)s', level=logging.WARNING)

    args = parse_args()

    args.target_dir = os.path.abspath(args.target_dir)
    search_path = [os.path.abspath(d) for d in args.include_dir]

    deps = []

    for src_rel in args.source:
        dest = os.path.join(args.target_dir, src_rel)
        deps.extend(copy_with_deps(src_rel, dest, search_path, args.include))

    deps = sorted(set(deps))

    write_depfile(args.depfile, args.target_name, deps)

if __name__ == "__main__":
    main()