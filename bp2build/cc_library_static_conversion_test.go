// Copyright 2021 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bp2build

import (
	"android/soong/android"
	"android/soong/cc"
	"strings"
	"testing"
)

const (
	// See cc/testing.go for more context
	soongCcLibraryStaticPreamble = `
cc_defaults {
	name: "linux_bionic_supported",
}

toolchain_library {
	name: "libclang_rt.builtins-x86_64-android",
	defaults: ["linux_bionic_supported"],
	vendor_available: true,
	vendor_ramdisk_available: true,
	product_available: true,
	recovery_available: true,
	native_bridge_supported: true,
	src: "",
}`
)

func TestCcLibraryStaticLoadStatement(t *testing.T) {
	testCases := []struct {
		bazelTargets           BazelTargets
		expectedLoadStatements string
	}{
		{
			bazelTargets: BazelTargets{
				BazelTarget{
					name:      "cc_library_static_target",
					ruleClass: "cc_library_static",
					// NOTE: No bzlLoadLocation for native rules
				},
			},
			expectedLoadStatements: ``,
		},
	}

	for _, testCase := range testCases {
		actual := testCase.bazelTargets.LoadStatements()
		expected := testCase.expectedLoadStatements
		if actual != expected {
			t.Fatalf("Expected load statements to be %s, got %s", expected, actual)
		}
	}

}

func TestCcLibraryStaticBp2Build(t *testing.T) {
	testCases := []struct {
		description                        string
		moduleTypeUnderTest                string
		moduleTypeUnderTestFactory         android.ModuleFactory
		moduleTypeUnderTestBp2BuildMutator func(android.TopDownMutatorContext)
		preArchMutators                    []android.RegisterMutatorFunc
		depsMutators                       []android.RegisterMutatorFunc
		bp                                 string
		expectedBazelTargets               []string
		filesystem                         map[string]string
		dir                                string
	}{
		{
			description:                        "cc_library_static test",
			moduleTypeUnderTest:                "cc_library_static",
			moduleTypeUnderTestFactory:         cc.LibraryStaticFactory,
			moduleTypeUnderTestBp2BuildMutator: cc.CcLibraryStaticBp2Build,
			filesystem: map[string]string{
				// NOTE: include_dir headers *should not* appear in Bazel hdrs later (?)
				"include_dir_1/include_dir_1_a.h": "",
				"include_dir_1/include_dir_1_b.h": "",
				"include_dir_2/include_dir_2_a.h": "",
				"include_dir_2/include_dir_2_b.h": "",
				// NOTE: local_include_dir headers *should not* appear in Bazel hdrs later (?)
				"local_include_dir_1/local_include_dir_1_a.h": "",
				"local_include_dir_1/local_include_dir_1_b.h": "",
				"local_include_dir_2/local_include_dir_2_a.h": "",
				"local_include_dir_2/local_include_dir_2_b.h": "",
				// NOTE: export_include_dir headers *should* appear in Bazel hdrs later
				"export_include_dir_1/export_include_dir_1_a.h": "",
				"export_include_dir_1/export_include_dir_1_b.h": "",
				"export_include_dir_2/export_include_dir_2_a.h": "",
				"export_include_dir_2/export_include_dir_2_b.h": "",
				// NOTE: Soong implicitly includes headers in the current directory
				"implicit_include_1.h": "",
				"implicit_include_2.h": "",
			},
			bp: soongCcLibraryStaticPreamble + `
cc_library_headers {
    name: "header_lib_1",
    export_include_dirs: ["header_lib_1"],
}

cc_library_headers {
    name: "header_lib_2",
    export_include_dirs: ["header_lib_2"],
}

cc_library_static {
    name: "static_lib_1",
    srcs: ["static_lib_1.cc"],
}

cc_library_static {
    name: "static_lib_2",
    srcs: ["static_lib_2.cc"],
}

cc_library_static {
    name: "whole_static_lib_1",
    srcs: ["whole_static_lib_1.cc"],
}

cc_library_static {
    name: "whole_static_lib_2",
    srcs: ["whole_static_lib_2.cc"],
}

cc_library_static {
    name: "foo_static",
    srcs: [
        "foo_static1.cc",
        "foo_static2.cc",
    ],
    cflags: [
        "-Dflag1",
        "-Dflag2"
    ],
    static_libs: [
        "static_lib_1",
        "static_lib_2"
    ],
    whole_static_libs: [
        "whole_static_lib_1",
        "whole_static_lib_2"
    ],
    include_dirs: [
        "include_dir_1",
        "include_dir_2",
    ],
    local_include_dirs: [
        "local_include_dir_1",
        "local_include_dir_2",
    ],
    export_include_dirs: [
    "export_include_dir_1",
    "export_include_dir_2"
    ],
    header_libs: [
        "header_lib_1",
        "header_lib_2"
    ],

    // TODO: Also support export_header_lib_headers
}`,
			expectedBazelTargets: []string{`cc_library_static(
    name = "foo_static",
    copts = [
        "-Dflag1",
        "-Dflag2",
        "-Iinclude_dir_1",
        "-Iinclude_dir_2",
        "-Ilocal_include_dir_1",
        "-Ilocal_include_dir_2",
        "-I.",
    ],
    deps = [
        ":header_lib_1",
        ":header_lib_2",
        ":static_lib_1",
        ":static_lib_2",
        ":whole_static_lib_1",
        ":whole_static_lib_2",
    ],
    hdrs = [
        "export_include_dir_1/export_include_dir_1_a.h",
        "export_include_dir_1/export_include_dir_1_b.h",
        "export_include_dir_2/export_include_dir_2_a.h",
        "export_include_dir_2/export_include_dir_2_b.h",
    ],
    includes = [
        "export_include_dir_1",
        "export_include_dir_2",
    ],
    linkstatic = True,
    srcs = [
        "foo_static1.cc",
        "foo_static2.cc",
        "implicit_include_1.h",
        "implicit_include_2.h",
        "export_include_dir_1/export_include_dir_1_a.h",
        "export_include_dir_1/export_include_dir_1_b.h",
        "export_include_dir_2/export_include_dir_2_a.h",
        "export_include_dir_2/export_include_dir_2_b.h",
        "include_dir_1/include_dir_1_a.h",
        "include_dir_1/include_dir_1_b.h",
        "include_dir_2/include_dir_2_a.h",
        "include_dir_2/include_dir_2_b.h",
        "local_include_dir_1/local_include_dir_1_a.h",
        "local_include_dir_1/local_include_dir_1_b.h",
        "local_include_dir_2/local_include_dir_2_a.h",
        "local_include_dir_2/local_include_dir_2_b.h",
    ],
)`, `cc_library_static(
    name = "static_lib_1",
    copts = ["-I."],
    linkstatic = True,
    srcs = [
        "static_lib_1.cc",
        "implicit_include_1.h",
        "implicit_include_2.h",
        "export_include_dir_1/export_include_dir_1_a.h",
        "export_include_dir_1/export_include_dir_1_b.h",
        "export_include_dir_2/export_include_dir_2_a.h",
        "export_include_dir_2/export_include_dir_2_b.h",
        "include_dir_1/include_dir_1_a.h",
        "include_dir_1/include_dir_1_b.h",
        "include_dir_2/include_dir_2_a.h",
        "include_dir_2/include_dir_2_b.h",
        "local_include_dir_1/local_include_dir_1_a.h",
        "local_include_dir_1/local_include_dir_1_b.h",
        "local_include_dir_2/local_include_dir_2_a.h",
        "local_include_dir_2/local_include_dir_2_b.h",
    ],
)`, `cc_library_static(
    name = "static_lib_2",
    copts = ["-I."],
    linkstatic = True,
    srcs = [
        "static_lib_2.cc",
        "implicit_include_1.h",
        "implicit_include_2.h",
        "export_include_dir_1/export_include_dir_1_a.h",
        "export_include_dir_1/export_include_dir_1_b.h",
        "export_include_dir_2/export_include_dir_2_a.h",
        "export_include_dir_2/export_include_dir_2_b.h",
        "include_dir_1/include_dir_1_a.h",
        "include_dir_1/include_dir_1_b.h",
        "include_dir_2/include_dir_2_a.h",
        "include_dir_2/include_dir_2_b.h",
        "local_include_dir_1/local_include_dir_1_a.h",
        "local_include_dir_1/local_include_dir_1_b.h",
        "local_include_dir_2/local_include_dir_2_a.h",
        "local_include_dir_2/local_include_dir_2_b.h",
    ],
)`, `cc_library_static(
    name = "whole_static_lib_1",
    copts = ["-I."],
    linkstatic = True,
    srcs = [
        "whole_static_lib_1.cc",
        "implicit_include_1.h",
        "implicit_include_2.h",
        "export_include_dir_1/export_include_dir_1_a.h",
        "export_include_dir_1/export_include_dir_1_b.h",
        "export_include_dir_2/export_include_dir_2_a.h",
        "export_include_dir_2/export_include_dir_2_b.h",
        "include_dir_1/include_dir_1_a.h",
        "include_dir_1/include_dir_1_b.h",
        "include_dir_2/include_dir_2_a.h",
        "include_dir_2/include_dir_2_b.h",
        "local_include_dir_1/local_include_dir_1_a.h",
        "local_include_dir_1/local_include_dir_1_b.h",
        "local_include_dir_2/local_include_dir_2_a.h",
        "local_include_dir_2/local_include_dir_2_b.h",
    ],
)`, `cc_library_static(
    name = "whole_static_lib_2",
    copts = ["-I."],
    linkstatic = True,
    srcs = [
        "whole_static_lib_2.cc",
        "implicit_include_1.h",
        "implicit_include_2.h",
        "export_include_dir_1/export_include_dir_1_a.h",
        "export_include_dir_1/export_include_dir_1_b.h",
        "export_include_dir_2/export_include_dir_2_a.h",
        "export_include_dir_2/export_include_dir_2_b.h",
        "include_dir_1/include_dir_1_a.h",
        "include_dir_1/include_dir_1_b.h",
        "include_dir_2/include_dir_2_a.h",
        "include_dir_2/include_dir_2_b.h",
        "local_include_dir_1/local_include_dir_1_a.h",
        "local_include_dir_1/local_include_dir_1_b.h",
        "local_include_dir_2/local_include_dir_2_a.h",
        "local_include_dir_2/local_include_dir_2_b.h",
    ],
)`},
		},
		{
			description:                        "cc_library_static subpackage test",
			moduleTypeUnderTest:                "cc_library_static",
			moduleTypeUnderTestFactory:         cc.LibraryStaticFactory,
			moduleTypeUnderTestBp2BuildMutator: cc.CcLibraryStaticBp2Build,
			filesystem: map[string]string{
				// subpackage with subdirectory
				"subpackage/Android.bp":                         "",
				"subpackage/subpackage_header.h":                "",
				"subpackage/subdirectory/subdirectory_header.h": "",
				// subsubpackage with subdirectory
				"subpackage/subsubpackage/Android.bp":                         "",
				"subpackage/subsubpackage/subsubpackage_header.h":             "",
				"subpackage/subsubpackage/subdirectory/subdirectory_header.h": "",
				// subsubsubpackage with subdirectory
				"subpackage/subsubpackage/subsubsubpackage/Android.bp":                         "",
				"subpackage/subsubpackage/subsubsubpackage/subsubsubpackage_header.h":          "",
				"subpackage/subsubpackage/subsubsubpackage/subdirectory/subdirectory_header.h": "",
			},
			bp: soongCcLibraryStaticPreamble + `
cc_library_static {
    name: "foo_static",
    srcs: [
    ],
    include_dirs: [
	"subpackage",
    ],
}`,
			expectedBazelTargets: []string{`cc_library_static(
    name = "foo_static",
    copts = [
        "-Isubpackage",
        "-I.",
    ],
    linkstatic = True,
    srcs = [
        "//subpackage:subpackage_header.h",
        "//subpackage:subdirectory/subdirectory_header.h",
        "//subpackage/subsubpackage:subsubpackage_header.h",
        "//subpackage/subsubpackage:subdirectory/subdirectory_header.h",
        "//subpackage/subsubpackage/subsubsubpackage:subsubsubpackage_header.h",
        "//subpackage/subsubpackage/subsubsubpackage:subdirectory/subdirectory_header.h",
    ],
)`},
		},
		{
			description:                        "cc_library_static export include dir",
			moduleTypeUnderTest:                "cc_library_static",
			moduleTypeUnderTestFactory:         cc.LibraryStaticFactory,
			moduleTypeUnderTestBp2BuildMutator: cc.CcLibraryStaticBp2Build,
			filesystem: map[string]string{
				// subpackage with subdirectory
				"subpackage/Android.bp":                         "",
				"subpackage/subpackage_header.h":                "",
				"subpackage/subdirectory/subdirectory_header.h": "",
			},
			bp: soongCcLibraryStaticPreamble + `
cc_library_static {
    name: "foo_static",
    export_include_dirs: ["subpackage"],
}`,
			expectedBazelTargets: []string{`cc_library_static(
    name = "foo_static",
    copts = ["-I."],
    hdrs = [
        "//subpackage:subdirectory/subdirectory_header.h",
        "//subpackage:subpackage_header.h",
    ],
    includes = ["subpackage"],
    linkstatic = True,
    srcs = [
        "//subpackage:subpackage_header.h",
        "//subpackage:subdirectory/subdirectory_header.h",
    ],
)`},
		},
		{
			description:                        "cc_library_static export system include dir",
			moduleTypeUnderTest:                "cc_library_static",
			moduleTypeUnderTestFactory:         cc.LibraryStaticFactory,
			moduleTypeUnderTestBp2BuildMutator: cc.CcLibraryStaticBp2Build,
			filesystem: map[string]string{
				// subpackage with subdirectory
				"subpackage/Android.bp":                         "",
				"subpackage/subpackage_header.h":                "",
				"subpackage/subdirectory/subdirectory_header.h": "",
			},
			bp: soongCcLibraryStaticPreamble + `
cc_library_static {
    name: "foo_static",
    export_system_include_dirs: ["subpackage"],
}`,
			expectedBazelTargets: []string{`cc_library_static(
    name = "foo_static",
    copts = ["-I."],
    hdrs = [
        "//subpackage:subdirectory/subdirectory_header.h",
        "//subpackage:subpackage_header.h",
    ],
    includes = ["subpackage"],
    linkstatic = True,
    srcs = [
        "//subpackage:subpackage_header.h",
        "//subpackage:subdirectory/subdirectory_header.h",
    ],
)`},
		},
		{
			description:                        "cc_library_static include_dirs, local_include_dirs, export_include_dirs (b/183742505)",
			moduleTypeUnderTest:                "cc_library_static",
			moduleTypeUnderTestFactory:         cc.LibraryStaticFactory,
			moduleTypeUnderTestBp2BuildMutator: cc.CcLibraryStaticBp2Build,
			dir:                                "subpackage",
			filesystem: map[string]string{
				// subpackage with subdirectory
				"subpackage/Android.bp": `
cc_library_static {
    name: "foo_static",
    // include_dirs are workspace/root relative
    include_dirs: [
        "subpackage/subsubpackage",
        "subpackage2",
        "subpackage3/subsubpackage"
    ],
    local_include_dirs: ["subsubpackage2"], // module dir relative
    export_include_dirs: ["./exported_subsubpackage"], // module dir relative
    include_build_directory: true,
    bazel_module: { bp2build_available: true },
}`,
				"subpackage/subsubpackage/header.h":          "",
				"subpackage/subsubpackage2/header.h":         "",
				"subpackage/exported_subsubpackage/header.h": "",
				"subpackage2/header.h":                       "",
				"subpackage3/subsubpackage/header.h":         "",
			},
			bp: soongCcLibraryStaticPreamble,
			expectedBazelTargets: []string{`cc_library_static(
    name = "foo_static",
    copts = [
        "-Isubpackage/subsubpackage",
        "-Isubpackage2",
        "-Isubpackage3/subsubpackage",
        "-Isubpackage/subsubpackage2",
        "-Isubpackage",
    ],
    hdrs = ["exported_subsubpackage/header.h"],
    includes = ["./exported_subsubpackage"],
    linkstatic = True,
    srcs = [
        "exported_subsubpackage/header.h",
        "subsubpackage/header.h",
        "subsubpackage2/header.h",
    ],
)`},
		},
		{
			description:                        "cc_library_static include_build_directory disabled",
			moduleTypeUnderTest:                "cc_library_static",
			moduleTypeUnderTestFactory:         cc.LibraryStaticFactory,
			moduleTypeUnderTestBp2BuildMutator: cc.CcLibraryStaticBp2Build,
			filesystem: map[string]string{
				// subpackage with subdirectory
				"subpackage/Android.bp":                         "",
				"subpackage/subpackage_header.h":                "",
				"subpackage/subdirectory/subdirectory_header.h": "",
			},
			bp: soongCcLibraryStaticPreamble + `
cc_library_static {
    name: "foo_static",
    include_dirs: ["subpackage"], // still used, but local_include_dirs is recommended
    local_include_dirs: ["subpackage2"],
    include_build_directory: false,
}`,
			expectedBazelTargets: []string{`cc_library_static(
    name = "foo_static",
    copts = [
        "-Isubpackage",
        "-Isubpackage2",
    ],
    linkstatic = True,
)`},
		},
		{
			description:                        "cc_library_static include_build_directory enabled",
			moduleTypeUnderTest:                "cc_library_static",
			moduleTypeUnderTestFactory:         cc.LibraryStaticFactory,
			moduleTypeUnderTestBp2BuildMutator: cc.CcLibraryStaticBp2Build,
			filesystem: map[string]string{
				// subpackage with subdirectory
				"subpackage/Android.bp":                         "",
				"subpackage/subpackage_header.h":                "",
				"subpackage2/Android.bp":                        "",
				"subpackage2/subpackage2_header.h":              "",
				"subpackage/subdirectory/subdirectory_header.h": "",
			},
			bp: soongCcLibraryStaticPreamble + `
cc_library_static {
    name: "foo_static",
    include_dirs: ["subpackage"], // still used, but local_include_dirs is recommended
    local_include_dirs: ["subpackage2"],
    include_build_directory: true,
}`,
			expectedBazelTargets: []string{`cc_library_static(
    name = "foo_static",
    copts = [
        "-Isubpackage",
        "-Isubpackage2",
        "-I.",
    ],
    linkstatic = True,
    srcs = [
        "//subpackage:subpackage_header.h",
        "//subpackage:subdirectory/subdirectory_header.h",
        "//subpackage2:subpackage2_header.h",
    ],
)`},
		},
		{
			description:                        "cc_library_static arch-specific static_libs",
			moduleTypeUnderTest:                "cc_library_static",
			moduleTypeUnderTestFactory:         cc.LibraryStaticFactory,
			moduleTypeUnderTestBp2BuildMutator: cc.CcLibraryStaticBp2Build,
			depsMutators:                       []android.RegisterMutatorFunc{cc.RegisterDepsBp2Build},
			filesystem:                         map[string]string{},
			bp: soongCcLibraryStaticPreamble + `
cc_library_static { name: "static_dep" }
cc_library_static { name: "static_dep2" }
cc_library_static {
    name: "foo_static",
    arch: { arm64: { static_libs: ["static_dep"], whole_static_libs: ["static_dep2"] } },
}`,
			expectedBazelTargets: []string{`cc_library_static(
    name = "foo_static",
    copts = ["-I."],
    deps = select({
        "//build/bazel/platforms/arch:arm64": [
            ":static_dep",
            ":static_dep2",
        ],
        "//conditions:default": [],
    }),
    linkstatic = True,
)`, `cc_library_static(
    name = "static_dep",
    copts = ["-I."],
    linkstatic = True,
)`, `cc_library_static(
    name = "static_dep2",
    copts = ["-I."],
    linkstatic = True,
)`},
		},
		{
			description:                        "cc_library_static os-specific static_libs",
			moduleTypeUnderTest:                "cc_library_static",
			moduleTypeUnderTestFactory:         cc.LibraryStaticFactory,
			moduleTypeUnderTestBp2BuildMutator: cc.CcLibraryStaticBp2Build,
			depsMutators:                       []android.RegisterMutatorFunc{cc.RegisterDepsBp2Build},
			filesystem:                         map[string]string{},
			bp: soongCcLibraryStaticPreamble + `
cc_library_static { name: "static_dep" }
cc_library_static { name: "static_dep2" }
cc_library_static {
    name: "foo_static",
    target: { android: { static_libs: ["static_dep"], whole_static_libs: ["static_dep2"] } },
}`,
			expectedBazelTargets: []string{`cc_library_static(
    name = "foo_static",
    copts = ["-I."],
    deps = select({
        "//build/bazel/platforms/os:android": [
            ":static_dep",
            ":static_dep2",
        ],
        "//conditions:default": [],
    }),
    linkstatic = True,
)`, `cc_library_static(
    name = "static_dep",
    copts = ["-I."],
    linkstatic = True,
)`, `cc_library_static(
    name = "static_dep2",
    copts = ["-I."],
    linkstatic = True,
)`},
		},
		{
			description:                        "cc_library_static base, arch and os-specific static_libs",
			moduleTypeUnderTest:                "cc_library_static",
			moduleTypeUnderTestFactory:         cc.LibraryStaticFactory,
			moduleTypeUnderTestBp2BuildMutator: cc.CcLibraryStaticBp2Build,
			depsMutators:                       []android.RegisterMutatorFunc{cc.RegisterDepsBp2Build},
			filesystem:                         map[string]string{},
			bp: soongCcLibraryStaticPreamble + `
cc_library_static { name: "static_dep" }
cc_library_static { name: "static_dep2" }
cc_library_static { name: "static_dep3" }
cc_library_static { name: "static_dep4" }
cc_library_static {
    name: "foo_static",
    static_libs: ["static_dep"],
    whole_static_libs: ["static_dep2"],
    target: { android: { static_libs: ["static_dep3"] } },
    arch: { arm64: { static_libs: ["static_dep4"] } },
}`,
			expectedBazelTargets: []string{`cc_library_static(
    name = "foo_static",
    copts = ["-I."],
    deps = [
        ":static_dep",
        ":static_dep2",
    ] + select({
        "//build/bazel/platforms/arch:arm64": [":static_dep4"],
        "//conditions:default": [],
    }) + select({
        "//build/bazel/platforms/os:android": [":static_dep3"],
        "//conditions:default": [],
    }),
    linkstatic = True,
)`, `cc_library_static(
    name = "static_dep",
    copts = ["-I."],
    linkstatic = True,
)`, `cc_library_static(
    name = "static_dep2",
    copts = ["-I."],
    linkstatic = True,
)`, `cc_library_static(
    name = "static_dep3",
    copts = ["-I."],
    linkstatic = True,
)`, `cc_library_static(
    name = "static_dep4",
    copts = ["-I."],
    linkstatic = True,
)`},
		},
	}

	dir := "."
	for _, testCase := range testCases {
		filesystem := make(map[string][]byte)
		toParse := []string{
			"Android.bp",
		}
		for f, content := range testCase.filesystem {
			if strings.HasSuffix(f, "Android.bp") {
				toParse = append(toParse, f)
			}
			filesystem[f] = []byte(content)
		}
		config := android.TestConfig(buildDir, nil, testCase.bp, filesystem)
		ctx := android.NewTestContext(config)

		cc.RegisterCCBuildComponents(ctx)
		ctx.RegisterModuleType("toolchain_library", cc.ToolchainLibraryFactory)
		ctx.RegisterModuleType("cc_library_headers", cc.LibraryHeaderFactory)

		ctx.RegisterModuleType(testCase.moduleTypeUnderTest, testCase.moduleTypeUnderTestFactory)
		for _, m := range testCase.depsMutators {
			ctx.DepsBp2BuildMutators(m)
		}
		ctx.RegisterBp2BuildMutator(testCase.moduleTypeUnderTest, testCase.moduleTypeUnderTestBp2BuildMutator)
		ctx.RegisterBp2BuildConfig(bp2buildConfig)
		ctx.RegisterForBazelConversion()

		_, errs := ctx.ParseFileList(dir, toParse)
		if Errored(t, testCase.description, errs) {
			continue
		}
		_, errs = ctx.ResolveDependencies(config)
		if Errored(t, testCase.description, errs) {
			continue
		}

		checkDir := dir
		if testCase.dir != "" {
			checkDir = testCase.dir
		}
		codegenCtx := NewCodegenContext(config, *ctx.Context, Bp2Build)
		bazelTargets := generateBazelTargetsForDir(codegenCtx, checkDir)
		if actualCount, expectedCount := len(bazelTargets), len(testCase.expectedBazelTargets); actualCount != expectedCount {
			t.Errorf("%s: Expected %d bazel target, got %d", testCase.description, expectedCount, actualCount)
		} else {
			for i, target := range bazelTargets {
				if w, g := testCase.expectedBazelTargets[i], target.content; w != g {
					t.Errorf(
						"%s: Expected generated Bazel target to be '%s', got '%s'",
						testCase.description,
						w,
						g,
					)
				}
			}
		}
	}
}
