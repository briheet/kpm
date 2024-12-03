package client

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/otiai10/copy"
	"github.com/stretchr/testify/assert"
	"kcl-lang.io/kpm/pkg/downloader"
	"kcl-lang.io/kpm/pkg/features"
	pkg "kcl-lang.io/kpm/pkg/package"
	"kcl-lang.io/kpm/pkg/utils"
)

func TestKclIssue1760(t *testing.T) {
	testPath := "github.com/kcl-lang/kcl/issues/1760"
	testCases := []struct {
		name  string
		setup func()
	}{
		{
			name: "Default",
			setup: func() {
				features.Disable(features.SupportNewStorage)
				features.Disable(features.SupportMVS)
			},
		},
		{
			name: "SupportNewStorage",
			setup: func() {
				features.Enable(features.SupportNewStorage)
				features.Disable(features.SupportMVS)
			},
		},
		{
			name: "SupportMVS",
			setup: func() {
				features.Disable(features.SupportNewStorage)
				features.Enable(features.SupportMVS)
			},
		},
		{
			name: "SupportNewStorageAndMVS",
			setup: func() {
				features.Enable(features.SupportNewStorage)
				features.Enable(features.SupportMVS)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc // capture range variable
		tc.setup()

		testFunc := func(t *testing.T, kpmcli *KpmClient) {
			rootPath := getTestDir("issues")
			mainKFilePath := filepath.Join(rootPath, testPath, "a", "main.k")
			var buf bytes.Buffer
			kpmcli.SetLogWriter(&buf)

			res, err := kpmcli.Run(
				WithRunSource(
					&downloader.Source{
						Local: &downloader.Local{
							Path: mainKFilePath,
						},
					},
				),
			)

			if err != nil {
				t.Fatal(err)
			}

			assert.Contains(t,
				utils.RmNewline(buf.String()),
				"downloading 'kcl-lang/fluxcd-source-controller:v1.3.2' from 'ghcr.io/kcl-lang/fluxcd-source-controller:v1.3.2'",
			)
			assert.Contains(t,
				utils.RmNewline(buf.String()),
				"downloading 'kcl-lang/k8s:1.31.2' from 'ghcr.io/kcl-lang/k8s:1.31.2'",
			)

			assert.Contains(t,
				utils.RmNewline(buf.String()),
				"downloading 'kcl-lang/fluxcd-helm-controller:v1.0.3' from 'ghcr.io/kcl-lang/fluxcd-helm-controller:v1.0.3'",
			)
			assert.Equal(t, res.GetRawYamlResult(), "The_first_kcl_program: Hello World!")
		}

		RunTestWithGlobalLockAndKpmCli(t, []TestSuite{{Name: tc.name, TestFunc: testFunc}})
	}
}

func TestKpmIssue550(t *testing.T) {
	testPath := "github.com/kcl-lang/kpm/issues/550"
	testCases := []struct {
		name        string
		setup       func()
		expected    string
		winExpected string
	}{
		{
			name: "Default",
			setup: func() {
				features.Disable(features.SupportNewStorage)
				features.Disable(features.SupportMVS)
			},
			expected:    filepath.Join("flask-demo-kcl-manifests_test-branch-without-modfile", "aa", "cc"),
			winExpected: filepath.Join("flask-demo-kcl-manifests_test-branch-without-modfile", "aa", "cc"),
		},
		{
			name: "SupportNewStorage",
			setup: func() {
				features.Enable(features.SupportNewStorage)
				features.Disable(features.SupportMVS)
			},
			expected:    filepath.Join("git", "src", "200297ed26e4aeb7", "flask-demo-kcl-manifests", "test-branch-without-modfile", "aa", "cc"),
			winExpected: filepath.Join("git", "src", "3523a44a55384201", "flask-demo-kcl-manifests", "test-branch-without-modfile", "aa", "cc"),
		},
		{
			name: "SupportMVS",
			setup: func() {
				features.Disable(features.SupportNewStorage)
				features.Enable(features.SupportMVS)
			},
			expected:    filepath.Join("flask-demo-kcl-manifests_test-branch-without-modfile", "aa", "cc"),
			winExpected: filepath.Join("flask-demo-kcl-manifests_test-branch-without-modfile", "aa", "cc"),
		},
		{
			name: "SupportNewStorageAndMVS",
			setup: func() {
				features.Enable(features.SupportNewStorage)
				features.Enable(features.SupportMVS)
			},
			expected:    filepath.Join("git", "src", "200297ed26e4aeb7", "flask-demo-kcl-manifests", "test-branch-without-modfile", "aa", "cc"),
			winExpected: filepath.Join("git", "src", "3523a44a55384201", "flask-demo-kcl-manifests", "test-branch-without-modfile", "aa", "cc"),
		},
	}

	for _, tc := range testCases {
		tc := tc // capture range variable

		tc.setup()

		testFunc := func(t *testing.T, kpmcli *KpmClient) {
			rootPath := getTestDir("issues")
			modPath := filepath.Join(rootPath, testPath, "pkg")
			var buf bytes.Buffer
			kpmcli.SetLogWriter(&buf)

			tmpKpmHome, err := os.MkdirTemp("", "")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpKpmHome)

			kpmcli.homePath = tmpKpmHome

			kMod, err := pkg.LoadKclPkgWithOpts(
				pkg.WithPath(modPath),
			)

			if err != nil {
				t.Fatal(err)
			}

			res, err := kpmcli.ResolveDepsMetadataInJsonStr(kMod, true)

			if err != nil {
				t.Fatal(err)
			}

			expectedPath := filepath.Join(tmpKpmHome, tc.expected)
			if runtime.GOOS == "windows" {
				expectedPath = filepath.Join(tmpKpmHome, tc.winExpected)
				expectedPath = strings.ReplaceAll(expectedPath, "\\", "\\\\")
			}

			assert.Equal(t, res, fmt.Sprintf(
				`{"packages":{"cc":{"name":"cc","manifest_path":"%s"}}}`,
				expectedPath,
			))

			resMap, err := kpmcli.ResolveDepsIntoMap(kMod)

			if err != nil {
				t.Fatal(err)
			}
			fmt.Printf("buf.String(): %v\n", buf.String())
			assert.Contains(t,
				utils.RmNewline(buf.String()),
				"cloning 'https://github.com/kcl-lang/flask-demo-kcl-manifests.git' with branch 'test-branch-without-modfile'",
			)
			assert.Equal(t, len(resMap), 1)
			if runtime.GOOS == "windows" {
				assert.Equal(t, resMap["cc"], filepath.Join(tmpKpmHome, tc.winExpected))
			} else {
				assert.Equal(t, resMap["cc"], filepath.Join(tmpKpmHome, tc.expected))
			}
		}

		RunTestWithGlobalLockAndKpmCli(t, []TestSuite{{Name: tc.name, TestFunc: testFunc}})
	}
}

func TestKpmIssue226(t *testing.T) {
	testPath := "github.com/kcl-lang/kpm/issues/226"
	test_add_dep_with_git_commit := func(t *testing.T, kpmcli *KpmClient) {
		rootPath := getTestDir("issues")
		modPath := filepath.Join(rootPath, testPath, "add_with_commit")
		modFileBk := filepath.Join(modPath, "kcl.mod.bk")
		LockFileBk := filepath.Join(modPath, "kcl.mod.lock.bk")
		modFile := filepath.Join(modPath, "kcl.mod")
		LockFile := filepath.Join(modPath, "kcl.mod.lock")
		modFileExpect := filepath.Join(modPath, "kcl.mod.expect")
		LockFileExpect := filepath.Join(modPath, "kcl.mod.lock.expect")

		defer func() {
			_ = os.RemoveAll(modFile)
			_ = os.RemoveAll(LockFile)
		}()

		err := copy.Copy(modFileBk, modFile)
		if err != nil {
			t.Fatal(err)
		}
		err = copy.Copy(LockFileBk, LockFile)
		if err != nil {
			t.Fatal(err)
		}

		var buf bytes.Buffer
		kpmcli.SetLogWriter(&buf)

		kpkg, err := pkg.LoadKclPkgWithOpts(
			pkg.WithPath(modPath),
		)

		if err != nil {
			t.Fatal(err)
		}

		err = kpmcli.Add(
			WithAddKclPkg(kpkg),
			WithAddSource(
				&downloader.Source{
					Git: &downloader.Git{
						Url:    "https://github.com/kcl-lang/flask-demo-kcl-manifests.git",
						Commit: "ade147b",
					},
				},
			),
		)

		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, utils.RmNewline(buf.String()),
			"cloning 'https://github.com/kcl-lang/flask-demo-kcl-manifests.git' with commit 'ade147b'"+
				"adding dependency 'flask_manifests'"+
				"add dependency 'flask_manifests:0.0.1' successfully")

		modFileContent, err := os.ReadFile(modFile)
		if err != nil {
			t.Fatal(err)
		}
		lockFileContent, err := os.ReadFile(LockFile)
		if err != nil {
			t.Fatal(err)
		}

		modFileExpectContent, err := os.ReadFile(modFileExpect)
		if err != nil {
			t.Fatal(err)
		}

		lockFileExpectContent, err := os.ReadFile(LockFileExpect)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, utils.RmNewline(string(modFileContent)), utils.RmNewline(string(modFileExpectContent)))
		assert.Equal(t, utils.RmNewline(string(lockFileContent)), utils.RmNewline(string(lockFileExpectContent)))
	}

	test_update_with_git_commit := func(t *testing.T, kpmcli *KpmClient) {
		rootPath := getTestDir("issues")
		modPath := filepath.Join(rootPath, testPath, "update_check_version")
		modFileBk := filepath.Join(modPath, "kcl.mod.bk")
		LockFileBk := filepath.Join(modPath, "kcl.mod.lock.bk")
		modFile := filepath.Join(modPath, "kcl.mod")
		LockFile := filepath.Join(modPath, "kcl.mod.lock")
		modFileExpect := filepath.Join(modPath, "kcl.mod.expect")
		LockFileExpect := filepath.Join(modPath, "kcl.mod.lock.expect")

		defer func() {
			_ = os.RemoveAll(modFile)
			_ = os.RemoveAll(LockFile)
		}()

		err := copy.Copy(modFileBk, modFile)
		if err != nil {
			t.Fatal(err)
		}
		err = copy.Copy(LockFileBk, LockFile)
		if err != nil {
			t.Fatal(err)
		}

		var buf bytes.Buffer
		kpmcli.SetLogWriter(&buf)

		kpkg, err := pkg.LoadKclPkgWithOpts(
			pkg.WithPath(modPath),
		)

		if err != nil {
			t.Fatal(err)
		}

		_, err = kpmcli.Update(
			WithUpdatedKclPkg(kpkg),
		)

		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, utils.RmNewline(buf.String()),
			"cloning 'https://github.com/kcl-lang/flask-demo-kcl-manifests.git' with commit 'ade147b'")

		modFileContent, err := os.ReadFile(modFile)
		if err != nil {
			t.Fatal(err)
		}
		lockFileContent, err := os.ReadFile(LockFile)
		if err != nil {
			t.Fatal(err)
		}

		modFileExpectContent, err := os.ReadFile(modFileExpect)
		if err != nil {
			t.Fatal(err)
		}

		lockFileExpectContent, err := os.ReadFile(LockFileExpect)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, utils.RmNewline(string(modFileContent)), utils.RmNewline(string(modFileExpectContent)))
		assert.Equal(t, utils.RmNewline(string(lockFileContent)), utils.RmNewline(string(lockFileExpectContent)))
	}

	test_update_with_git_commit_invalid := func(t *testing.T, kpmcli *KpmClient) {
		rootPath := getTestDir("issues")
		modPath := filepath.Join(rootPath, testPath, "update_check_version_invalid")
		modFileBk := filepath.Join(modPath, "kcl.mod.bk")
		LockFileBk := filepath.Join(modPath, "kcl.mod.lock.bk")
		modFile := filepath.Join(modPath, "kcl.mod")
		LockFile := filepath.Join(modPath, "kcl.mod.lock")
		modFileExpect := filepath.Join(modPath, "kcl.mod.expect")
		LockFileExpect := filepath.Join(modPath, "kcl.mod.lock.expect")

		defer func() {
			_ = os.RemoveAll(modFile)
			_ = os.RemoveAll(LockFile)
		}()

		err := copy.Copy(modFileBk, modFile)
		if err != nil {
			t.Fatal(err)
		}
		err = copy.Copy(LockFileBk, LockFile)
		if err != nil {
			t.Fatal(err)
		}

		var buf bytes.Buffer
		kpmcli.SetLogWriter(&buf)

		kpkg, err := pkg.LoadKclPkgWithOpts(
			pkg.WithPath(modPath),
		)

		if err != nil {
			t.Fatal(err)
		}

		_, err = kpmcli.Update(
			WithUpdatedKclPkg(kpkg),
		)

		assert.Equal(t, err.Error(), "package 'flask_manifests:0.100.0' not found")

		assert.Equal(t, utils.RmNewline(buf.String()),
			"cloning 'https://github.com/kcl-lang/flask-demo-kcl-manifests.git' with commit 'ade147b'")

		modFileContent, err := os.ReadFile(modFile)
		if err != nil {
			t.Fatal(err)
		}
		lockFileContent, err := os.ReadFile(LockFile)
		if err != nil {
			t.Fatal(err)
		}

		modFileExpectContent, err := os.ReadFile(modFileExpect)
		if err != nil {
			t.Fatal(err)
		}

		lockFileExpectContent, err := os.ReadFile(LockFileExpect)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, utils.RmNewline(string(modFileContent)), utils.RmNewline(string(modFileExpectContent)))
		assert.Equal(t, utils.RmNewline(string(lockFileContent)), utils.RmNewline(string(lockFileExpectContent)))
	}

	RunTestWithGlobalLockAndKpmCli(t, []TestSuite{{Name: "add_dep_with_git_commit", TestFunc: test_add_dep_with_git_commit}})
	RunTestWithGlobalLockAndKpmCli(t, []TestSuite{{Name: "update_with_git_commit", TestFunc: test_update_with_git_commit}})
	RunTestWithGlobalLockAndKpmCli(t, []TestSuite{{Name: "update_with_git_commit_invalid", TestFunc: test_update_with_git_commit_invalid}})
}
