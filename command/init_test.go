package command

import (
	"io/ioutil"
	"log"
	"path/filepath"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/mod/sumdb/dirhash"
)

func TestInitCommand_Run(t *testing.T) {
	// These tests will try to optimise for doing the least amount of github api
	// requests whilst testing the max amount of things at once. Hopefully they
	// don't require a GH token just yet. Acc tests are run on linux, darwin and
	// windows, so requests are done 3 times.

	// if os.Getenv(acctest.TestEnvVar) == "" {
	// 	t.Skip(fmt.Sprintf("Acceptance tests skipped unless env '%s' set", acctest.TestEnvVar))
	// }

	type testCase struct {
		name                                  string
		inPluginFolder                        map[string]string
		expectedPackerConfigDirHashBeforeInit string
		hclFile                               string
		packerConfigDir                       string
		env                                   map[string]string
		want                                  int
		dirFiles                              []string
		expectedPackerConfigDirHashAfterInit  string
	}

	cfg := &configDirSingleton{map[string]string{}}

	tests := []testCase{
		{
			// here we pre-write plugins with valid checksums, Packer will
			// see those as valid installations it did.
			// the directory hash before and after init should be the same,
			// that's a no-op
			"already-installed-no-op",
			map[string]string{
				".plugin/github.com/sylviamoss/comment/packer-plugin-comment_v0.2.18_x5.0_darwin_amd64":            "1",
				".plugin/github.com/sylviamoss/comment/packer-plugin-comment_v0.2.18_x5.0_darwin_amd64_SHA256SUM":  "6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b",
				".plugin/github.com/sylviamoss/comment/packer-plugin-comment_v0.2.18_x5.0_windows_amd64":           "1.exe",
				".plugin/github.com/sylviamoss/comment/packer-plugin-comment_v0.2.18_x5.0_windows_amd64_SHA256SUM": "b238233f12d9d803d4df28ac0ce1e80ef93f66ea9391a25ac711a604168472bc",
				".plugin/github.com/sylviamoss/comment/packer-plugin-comment_v0.2.18_x5.0_linux_amd64":             "1.out",
				".plugin/github.com/sylviamoss/comment/packer-plugin-comment_v0.2.18_x5.0_linux_amd64_SHA256SUM":   "59031c50e0dfeedfde2b4e9445754804dce3f29e4efa737eead0ca9b4f5b85a5",
			},
			"h1:Jm/w2yUl6NzvyPQGcTJw7iOoEIWsZe9Pwb2/iBH39tA=",
			`# cfg.pkr.hcl
			packer {
				required_plugins {
					comment = {
						source  = "github.com/sylviamoss/comment"
						version = "v0.2.018"
					}
				}
			}`,
			cfg.dir("1"),
			map[string]string{
				"PACKER_CONFIG_DIR": cfg.dir("1"),
			},
			0,
			nil,
			"h1:Jm/w2yUl6NzvyPQGcTJw7iOoEIWsZe9Pwb2/iBH39tA=",
		},
		// {
		// 	// here we pre-write plugins with valid checksums, Packer will
		// 	// see those as valid installations it did.
		// 	// But because we require version 0.2.19, we will upgrade.
		// 	"already-installed-upgrade",
		// 	map[string]string{
		// 		".plugin/github.com/sylviamoss/comment/packer-plugin-comment_v0.2.18_x5.0_darwin_amd64":            "1",
		// 		".plugin/github.com/sylviamoss/comment/packer-plugin-comment_v0.2.18_x5.0_darwin_amd64_SHA256SUM":  "4355a46b19d348dc2f57c046f8ef63d4538ebb936000f3c9ee954a27460dd865",
		// 		".plugin/github.com/sylviamoss/comment/packer-plugin-comment_v0.2.18_x5.0_windows_amd64":           "1.exe",
		// 		".plugin/github.com/sylviamoss/comment/packer-plugin-comment_v0.2.18_x5.0_windows_amd64_SHA256SUM": "b238233f12d9d803d4df28ac0ce1e80ef93f66ea9391a25ac711a604168472bc",
		// 		".plugin/github.com/sylviamoss/comment/packer-plugin-comment_v0.2.18_x5.0_linux_amd64":             "1.out",
		// 		".plugin/github.com/sylviamoss/comment/packer-plugin-comment_v0.2.18_x5.0_linux_amd64_SHA256SUM":   "c28ae3482c9030519a9f5bdf6b3db4638076e6f99897e9b0e71bb38b0d76fd7e",
		// 	},
		// 	"h1:RgZ9LKqioZ4R+GN6oGXpDAMEKreMx1y9uFjyvzVRetI=",
		// 	`# cfg.pkr.hcl
		// 	packer {
		// 		required_plugins {
		// 			comment = {
		// 				source  = "github.com/sylviamoss/comment"
		// 				version = "v0.2.019"
		// 			}
		// 		}
		// 	}`,
		// 	cfg.dir("1"),
		// 	map[string]string{
		// 		"PACKER_CONFIG_DIR": cfg.dir("1"),
		// 	},
		// 	0,
		// 	[]string{
		// 		".plugin/github.com/sylviamoss/comment/packer-plugin-comment_v0.2.18_x5.0_darwin_amd64",
		// 		".plugin/github.com/sylviamoss/comment/packer-plugin-comment_v0.2.18_x5.0_darwin_amd64_SHA256SUM",
		// 		".plugin/github.com/sylviamoss/comment/packer-plugin-comment_v0.2.18_x5.0_linux_amd64",
		// 		".plugin/github.com/sylviamoss/comment/packer-plugin-comment_v0.2.18_x5.0_linux_amd64_SHA256SUM",
		// 		".plugin/github.com/sylviamoss/comment/packer-plugin-comment_v0.2.18_x5.0_windows_amd64",
		// 		".plugin/github.com/sylviamoss/comment/packer-plugin-comment_v0.2.18_x5.0_windows_amd64_SHA256SUM",
		// 		map[string]string{
		// 			"darwin":  ".plugin/github.com/sylviamoss/comment/packer-plugin-comment_v0.2.19_x5.0_darwin_amd64_SHA256SUM",
		// 			"linux":   ".plugin/github.com/sylviamoss/comment/packer-plugin-comment_v0.2.19_x5.0_linux_amd64_SHA256SUM",
		// 			"windows": ".plugin/github.com/sylviamoss/comment/packer-plugin-comment_v0.2.19_x5.0_windows_amd64.exe_SHA256SUM",
		// 		}[runtime.GOOS],
		// 		map[string]string{
		// 			"darwin":  ".plugin/github.com/sylviamoss/comment/packer-plugin-comment_v0.2.19_x5.0_darwin_amd64",
		// 			"linux":   ".plugin/github.com/sylviamoss/comment/packer-plugin-comment_v0.2.19_x5.0_linux_amd64",
		// 			"windows": ".plugin/github.com/sylviamoss/comment/packer-plugin-comment_v0.2.19_x5.0_windows_amd64.exe",
		// 		}[runtime.GOOS],
		// 	},
		// 	map[string]string{
		// 		"darwin":  "h1:wc1g0hs2FRoVXUsxzdcbQuWAvZw1wPRi79MFrlrJQjE=",
		// 		"linux":   "h1:RgZ9LKqioZ4R+GN6oGXpDAMEKreMx1y9uFjyvzVRetI=",
		// 		"windows": "h1:RgZ9LKqioZ4R+GN6oGXpDAMEKreMx1y9uFjyvzVRetI=",
		// 	}[runtime.GOOS],
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log.Printf("starting %s", tt.name)
			createFiles(tt.packerConfigDir, tt.inPluginFolder)

			hash, err := dirhash.HashDir(tt.packerConfigDir, "", dirhash.DefaultHash)
			if err != nil {
				t.Fatalf("HashDir: %v", err)
			}
			if diff := cmp.Diff(tt.expectedPackerConfigDirHashBeforeInit, hash); diff != "" {
				t.Errorf("unexpected dir hash before init: %s", diff)
			}

			cfgDir, err := ioutil.TempDir("", "pkr-test-init-file-folder")
			if err != nil {
				t.Fatalf("TempDir: %v", err)
			}
			if err := ioutil.WriteFile(filepath.Join(cfgDir, "cfg.pkr.hcl"), []byte(tt.hclFile), 0666); err != nil {
				t.Fatalf("WriteFile: %v", err)
			}

			args := []string{cfgDir}

			c := &InitCommand{
				Meta: testMetaFile(t),
			}
			c.CoreConfig.Components.PluginConfig.KnownPluginFolders = []string{filepath.Join(tt.packerConfigDir, ".plugin")}
			if got := c.Run(args); got != tt.want {
				t.Errorf("InitCommand.Run() = %v, want %v", got, tt.want)
			}

			if tt.dirFiles != nil {
				dirFiles, err := dirhash.DirFiles(tt.packerConfigDir, "")
				if err != nil {
					t.Fatalf("DirFiles: %v", err)
				}
				sort.Strings(tt.dirFiles)
				sort.Strings(dirFiles)
				if diff := cmp.Diff(tt.dirFiles, dirFiles); diff != "" {
					t.Errorf("found files differ: %v", diff)
				}
			}

			hash, err = dirhash.HashDir(tt.packerConfigDir, "", dirhash.DefaultHash)
			if err != nil {
				t.Fatalf("HashDir: %v", err)
			}
			if diff := cmp.Diff(tt.expectedPackerConfigDirHashAfterInit, hash); diff != "" {
				t.Errorf("unexpected dir hash after init: %s", diff)
			}
		})
	}
}
