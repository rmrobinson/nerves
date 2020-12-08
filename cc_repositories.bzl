load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

all_content = """filegroup(name = "all", srcs = glob(["**"]), visibility = ["//visibility:public"])"""

def include_cc_repositories():
    http_archive(
        name = "bottlerocket",
        build_file_content = """load("@rules_foreign_cc//tools/build_defs:configure.bzl", "configure_make")

filegroup(
    name = "sources",
    srcs = glob(["**"]),
)

configure_make(
    name = "libbr",
    configure_options = [
        "--with-x10port=/dev/firecracker",
    ],
    lib_source = ":sources",
    static_libraries = ["libbr.a"],
    make_commands = [
        "make",
        "make lib_install",
    ],
    visibility = ["//visibility:public"],
)
        """,
        sha256 = "ce26d6bba87244573b5b4db56f7acd969afb070fbd5ac364bd57201c0acb1267",
        strip_prefix = "bottlerocket-cleanup2",
        urls = [
            "https://github.com/rmrobinson/bottlerocket/archive/cleanup2.zip",
        ],
    )
