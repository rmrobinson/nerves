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
        sha256 = "fd5e0db1316f29586ed403ff8625f2f1313fd3a9e91d1b4476d18b6a5dba6287",
        strip_prefix = "bottlerocket-0.05b4",
        urls = [
            "https://github.com/rmrobinson/bottlerocket/archive/v0.05b4.zip",
        ],
    )
