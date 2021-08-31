// protobuf-compile is a helper tool for running protoc against all of the
// .proto files in this repository using specific versions of protoc and
// protoc-gen-go, to ensure consistent results across all development
// environments.
//
// protoc itself isn't a Go tool, so we need to use a custom strategy to
// install and run it. The official releases are built only for a subset of
// platforms that Go can potentially target, so this tool will fail if you
// are using a platform other than the ones this wrapper tool has explicit
// support for. In that case you'll need to either run this tool on a supported
// platform or to recreate what it does manually using a protoc you've built
// and installed yourself.
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hashicorp/go-getter"
)

const protocVersion = "3.15.6"

// We also use protoc-gen-go and its grpc addon, but since these are Go tools
// in Go modules our version selection for these comes from our top-level
// go.mod, as with all other Go dependencies. If you want to switch to a newer
// version of either tool then you can upgrade their modules in the usual way.
const protocGenGoPackage = "github.com/golang/protobuf/protoc-gen-go"
const protocGenGoGrpcPackage = "google.golang.org/grpc/cmd/protoc-gen-go-grpc"

type protocStep struct {
	DisplayName string
	WorkDir     string
	Args        []string
}

var protocSteps = []protocStep{
	{
		"tfplugin5 (provider wire protocol version 5)",
		"internal/tfplugin5",
		[]string{"--go_out=paths=source_relative,plugins=grpc:.", "./tfplugin5.proto"},
	},
	{
		"tfplugin6 (provider wire protocol version 6)",
		"internal/tfplugin6",
		[]string{"--go_out=paths=source_relative,plugins=grpc:.", "./tfplugin6.proto"},
	},
	{
		"tfplan (plan file serialization)",
		"internal/plans/internal/planproto",
		[]string{"--go_out=paths=source_relative:.", "planfile.proto"},
	},
}

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Usage: go run github.com/hashicorp/terraform/tools/protobuf-compile <basedir>")
	}
	baseDir := os.Args[1]
	workDir := filepath.Join(baseDir, "tools/protobuf-compile/.workdir")

	protocLocalDir := filepath.Join(workDir, "protoc-v"+protocVersion)
	if _, err := os.Stat(protocLocalDir); os.IsNotExist(err) {
		err := downloadProtoc(protocVersion, protocLocalDir)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		log.Printf("already have protoc v%s in %s", protocVersion, protocLocalDir)
	}

	protocExec := filepath.Join(protocLocalDir, "bin/protoc")

	protocGenGoExec, err := buildProtocGenGo(workDir)
	if err != nil {
		log.Fatal(err)
	}
	_, err = buildProtocGenGoGrpc(workDir)
	if err != nil {
		log.Fatal(err)
	}

	protocExec, err = filepath.Abs(protocExec)
	if err != nil {
		log.Fatal(err)
	}
	protocGenGoExec, err = filepath.Abs(protocGenGoExec)
	if err != nil {
		log.Fatal(err)
	}
	protocGenGoGrpcExec, err := filepath.Abs(protocGenGoExec)
	if err != nil {
		log.Fatal(err)
	}

	// For all of our steps we'll run our localized protoc with our localized
	// protoc-gen-go.
	baseCmdLine := []string{protocExec, "--plugin=" + protocGenGoExec, "--plugin=" + protocGenGoGrpcExec}

	for _, step := range protocSteps {
		log.Printf("working on %s", step.DisplayName)

		cmdLine := make([]string, 0, len(baseCmdLine)+len(step.Args))
		cmdLine = append(cmdLine, baseCmdLine...)
		cmdLine = append(cmdLine, step.Args...)

		cmd := &exec.Cmd{
			Path:   cmdLine[0],
			Args:   cmdLine[1:],
			Dir:    step.WorkDir,
			Env:    os.Environ(),
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		}
		err := cmd.Run()
		if err != nil {
			log.Printf("failed to compile: %s", err)
		}
	}

}

// downloadProtoc downloads the given version of protoc into the given
// directory.
func downloadProtoc(version string, localDir string) error {
	protocURL, err := protocDownloadURL(version)
	if err != nil {
		return err
	}

	log.Printf("downloading and extracting protoc v%s from %s into %s", version, protocURL, localDir)

	// For convenience, we'll be using go-getter to actually download this
	// thing, so we need to turn the real URL into the funny sort of pseudo-URL
	// thing that go-getter wants.
	goGetterURL := protocURL + "?archive=zip"

	err = getter.Get(localDir, goGetterURL)
	if err != nil {
		return fmt.Errorf("failed to download or extract the package: %s", err)
	}

	return nil
}

// buildProtocGenGo uses the Go toolchain to fetch the module containing
// protoc-gen-go and then build an executable into the working directory.
//
// If successful, it returns the location of the executable.
func buildProtocGenGo(workDir string) (string, error) {
	exeSuffixRaw, err := exec.Command("go", "env", "GOEXE").Output()
	if err != nil {
		return "", fmt.Errorf("failed to determine executable suffix: %s", err)
	}
	exeSuffix := strings.TrimSpace(string(exeSuffixRaw))
	exePath := filepath.Join(workDir, "protoc-gen-go"+exeSuffix)
	log.Printf("building %s as %s", protocGenGoPackage, exePath)

	cmd := exec.Command("go", "build", "-o", exePath, protocGenGoPackage)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to build %s: %s", protocGenGoPackage, err)
	}

	return exePath, nil
}

// buildProtocGenGoGrpc uses the Go toolchain to fetch the module containing
// protoc-gen-go-grpc and then build an executable into the working directory.
//
// If successful, it returns the location of the executable.
func buildProtocGenGoGrpc(workDir string) (string, error) {
	exeSuffixRaw, err := exec.Command("go", "env", "GOEXE").Output()
	if err != nil {
		return "", fmt.Errorf("failed to determine executable suffix: %s", err)
	}
	exeSuffix := strings.TrimSpace(string(exeSuffixRaw))
	exePath := filepath.Join(workDir, "protoc-gen-go-grpc"+exeSuffix)
	log.Printf("building %s as %s", protocGenGoGrpcPackage, exePath)

	cmd := exec.Command("go", "build", "-o", exePath, protocGenGoGrpcPackage)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to build %s: %s", protocGenGoGrpcPackage, err)
	}

	return exePath, nil
}

// protocDownloadURL returns the URL to try to download the protoc package
// for the current platform or an error if there's no known URL for the
// current platform.
func protocDownloadURL(version string) (string, error) {
	platformKW := protocPlatform()
	if platformKW == "" {
		return "", fmt.Errorf("don't know where to find protoc for %s on %s", runtime.GOOS, runtime.GOARCH)
	}
	return fmt.Sprintf("https://github.com/protocolbuffers/protobuf/releases/download/v%s/protoc-%s-%s.zip", protocVersion, protocVersion, platformKW), nil
}

// protocPlatform returns the package name substring for the current platform
// in the naming convention used by official protoc packages, or an empty
// string if we don't know how protoc packaging would describe current
// platform.
func protocPlatform() string {
	goPlatform := runtime.GOOS + "_" + runtime.GOARCH

	switch goPlatform {
	case "linux_amd64":
		return "linux-x86_64"
	case "linux_arm64":
		return "linux-aarch_64"
	case "darwin_amd64":
		return "osx-x86_64"
	case "darwin_arm64":
		// As of 3.15.6 there isn't yet an osx-aarch_64 package available,
		// so we'll install the x86_64 version and hope Rosetta can handle it.
		return "osx-x86_64"
	case "windows_amd64":
		return "win64" // for some reason the windows packages don't have a CPU architecture part
	default:
		return ""
	}
}
