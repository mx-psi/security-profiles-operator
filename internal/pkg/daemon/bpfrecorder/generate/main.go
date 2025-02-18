/*
Copyright 2021 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"sigs.k8s.io/security-profiles-operator/internal/pkg/daemon/bpfrecorder/types"
)

const header = `// +build linux

/*
Copyright 2021 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package bpfrecorder

`

const (
	buildDir    = "build/"
	bpfObj      = "recorder.bpf.o"
	baseDir     = "internal/pkg/daemon/bpfrecorder/"
	generatedGo = baseDir + "generated.go"
	btfDir      = baseDir + "btf"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	builder := &strings.Builder{}

	if err := generateBpfObj(builder); err != nil {
		return errors.Wrap(err, "generate bpf object")
	}

	if err := generateBtf(builder); err != nil {
		return errors.Wrap(err, "generate btf")
	}

	// nolint: gosec, gomnd
	if err := ioutil.WriteFile(
		generatedGo, []byte(builder.String()), 0o644,
	); err != nil {
		return errors.Wrap(err, "writing generated object")
	}
	if err := exec.Command("go", "fmt", generatedGo).Run(); err != nil {
		return errors.Wrap(err, "format go code")
	}
	return nil
}

func generateBpfObj(builder *strings.Builder) error {
	builder.WriteString(header)
	builder.WriteString("var bpfObjects = map[string][]byte{\n")

	for _, arch := range []string{"amd64", "arm64"} {
		builder.WriteString(fmt.Sprintf("%q: {\n", arch))

		file, err := ioutil.ReadFile(filepath.Join(buildDir, bpfObj+"."+arch))
		if err != nil {
			return errors.Wrap(err, "read bpf object path")
		}

		size := len(file)
		for k, v := range file {
			builder.WriteString(fmt.Sprint(v))

			if k < size-1 {
				builder.WriteString(", ")
			}

			if k != 0 && k%16 == 0 {
				builder.WriteString("\n\t")
			}
		}

		builder.WriteString("},\n")
	}

	builder.WriteString("}\n\n")
	return nil
}

func generateBtf(builder *strings.Builder) error {
	builder.WriteString("var btfJSON = `")
	btfs := types.Btf{}
	if err := filepath.Walk(btfDir, func(path string, info fs.FileInfo, retErr error) error {
		if info.IsDir() || filepath.Ext(path) != ".btf" {
			return nil
		}

		// A path should consist of:
		// - the btf dir
		// - the OS
		// - the OS version
		// - the architecture
		// - the btf file containing the kernel version
		pathSplit := strings.Split(path, string(os.PathSeparator))
		const expectedBPFPathLen = 9
		if len(pathSplit) != expectedBPFPathLen {
			return errors.Errorf("invalid btf path: %s (len = %d)", path, len(pathSplit))
		}

		btfBytes, err := ioutil.ReadFile(path)
		if err != nil {
			return errors.Wrap(err, "read btf file")
		}

		os := types.Os(pathSplit[5])
		osVersion := types.OsVersion(pathSplit[6])
		arch := types.Arch(pathSplit[7])
		kernel := types.Kernel(pathSplit[8][0 : len(pathSplit[8])-len(filepath.Ext(pathSplit[8]))])

		if _, ok := btfs[os]; !ok {
			btfs[os] = map[types.OsVersion]map[types.Arch]map[types.Kernel][]byte{}
		}
		if _, ok := btfs[os][osVersion]; !ok {
			btfs[os][osVersion] = map[types.Arch]map[types.Kernel][]byte{}
		}
		if _, ok := btfs[os][osVersion][arch]; !ok {
			btfs[os][osVersion][arch] = map[types.Kernel][]byte{}
		}

		btfs[os][osVersion][arch][kernel] = btfBytes

		return nil
	}); err != nil {
		return errors.Wrap(err, "walk btf files")
	}
	jsonBytes, err := json.MarshalIndent(btfs, "", "  ")
	if err != nil {
		return errors.Wrap(err, "marshal btf JSON")
	}
	builder.Write(jsonBytes)
	builder.WriteString("`\n")
	return nil
}
