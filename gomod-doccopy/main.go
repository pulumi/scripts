package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode"
)

var (
	flags             = flag.NewFlagSet("gomod-doccopy", flag.ExitOnError)
	providerSourceOrg = flags.String("src-org", "terraform-providers", "source provider GitHub org")
	providerDestOrg   = flags.String("dest-org", "terraform-providers", "source provider GitHub org")
	providerName      = flags.String("provider", "", "provider name")
	verbose           = flags.Bool("v", false, "verbose output")
)

func main() {
	flags.Parse(os.Args[1:])

	if *providerName == "" {
		fmt.Fprintf(os.Stderr, "missing required -provider flag value\n")
		os.Exit(1)
	}

	// Ensure go.mod file exists and we're running from the project root,
	// and that ./vendor/modules.txt file exists.
	cwd, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if _, err := os.Stat(filepath.Join(cwd, "go.mod")); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "cannot find `go.mod` file\n")
		os.Exit(1)
	}
	modtxtPath := filepath.Join(cwd, "vendor", "modules.txt")
	if _, err := os.Stat(modtxtPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "cannot find vendor/modules.txt, first run `go mod vendor` and try again\n")
		os.Exit(1)
	}

	targetProviderImportSourcePath := fmt.Sprintf("github.com/%s/%s", *providerSourceOrg, *providerName)
	targetProviderImportDestPath := fmt.Sprintf("vendor/github.com/%s/%s", *providerDestOrg, *providerName)
	fmt.Println(targetProviderImportSourcePath, " => ", targetProviderImportDestPath)

	// Parse/process modules.txt file of pkgs
	f, _ := os.Open(modtxtPath)
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		line := scanner.Text()

		if line[0] != 35 {
			continue
		}
		s := strings.Split(line, " ")

		if s[1] != targetProviderImportSourcePath {
			if *verbose == true {
				log.Printf("Ignoring import path: %s", s[1])
			}
			continue
		}

		moduleDirectory := pkgModPath(s[1], s[2])
		if *verbose == true {
			log.Printf("Needs to copy from %s", moduleDirectory)
		}

		if _, err := os.Stat(moduleDirectory); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "module path %q does not exist, check $GOPATH/pkg/mod\n", moduleDirectory)
			os.Exit(1)
		}

		src := moduleDirectory

		if err := os.RemoveAll(targetProviderImportDestPath); err != nil {
			fmt.Fprintf(os.Stderr, "error removing the target directory %q: %s\n", targetProviderImportDestPath, err)
			os.Exit(1)
		}

		if err := copyDir(src, targetProviderImportDestPath); err != nil {
			fmt.Fprintf(os.Stderr, "error copying provider directory: %s\n", err)
			os.Exit(1)
		}
	}
}

func pkgModPath(importPath, version string) string {
	goPath := os.Getenv("GOPATH")
	if goPath == "" {
		// the default GOPATH for go v1.11
		goPath = filepath.Join(os.Getenv("HOME"), "go")
	}

	var normPath string

	for _, char := range importPath {
		if unicode.IsUpper(char) {
			normPath += "!" + string(unicode.ToLower(char))
		} else {
			normPath += string(char)
		}
	}

	return filepath.Join(goPath, "pkg", "mod", fmt.Sprintf("%s@%s", normPath, version))
}

// Dir copies a whole directory recursively
func copyDir(src string, dst string) error {
	var err error
	var fds []os.FileInfo

	if err = os.MkdirAll(dst, 0744); err != nil {
		return err
	}

	if fds, err = ioutil.ReadDir(src); err != nil {
		return err
	}
	for _, fd := range fds {
		srcfp := path.Join(src, fd.Name())
		dstfp := path.Join(dst, fd.Name())

		if fd.IsDir() {
			if err = copyDir(srcfp, dstfp); err != nil {
				return err
			}
		} else {
			if err = copyFile(srcfp, dstfp); err != nil {
				return err
			}
		}
	}
	return nil
}

// File copies a single file from src to dst
func copyFile(src string, dst string) error {
	var err error
	var srcfd *os.File
	var dstfd *os.File

	if srcfd, err = os.Open(src); err != nil {
		return err
	}
	defer srcfd.Close()

	if dstfd, err = os.Create(dst); err != nil {
		return err
	}
	defer dstfd.Close()

	if _, err = io.Copy(dstfd, srcfd); err != nil {
		return err
	}
	return os.Chmod(dst, 0644)
}
