package doc

import (
	"go/ast"
	"go/token"
	"path/filepath"
	"strings"
)

type PackagesDefinitions struct {
	files    map[*ast.File]*AstFileInfo
	packages map[string]*PackageDefinitions
	excludes map[string]bool
	workdir  string
}

type Option func(pkg *PackagesDefinitions)

func NewPackagesDefinitions(opt ...Option) *PackagesDefinitions {
	pkg := &PackagesDefinitions{
		files:    make(map[*ast.File]*AstFileInfo),
		packages: make(map[string]*PackageDefinitions),
	}

	for _, o := range opt {
		o(pkg)
	}

	return pkg
}

func WithExcludes(dir map[string]bool) Option {
	return func(pkg *PackagesDefinitions) {
		pkg.excludes = dir
	}
}

func WithWorkdir(dir string) Option {
	return func(pkg *PackagesDefinitions) {
		pkg.workdir = dir
	}
}

func (pkgs *PackagesDefinitions) RangeFileForInjectTag(dryRun bool, filepath, propertyStrategy string, tagFlags []string, typeMap map[string]bool, handle func(dryRun bool, propertyStrategy string, tagFlags []string, typeMap map[string]bool, info *AstFileInfo, file *ast.File) error) error {
	for file, info := range pkgs.files {
		if info.Path == filepath {
			return handle(dryRun, propertyStrategy, tagFlags, typeMap, info, file)
		}
	}

	return nil
}

func (pkgs *PackagesDefinitions) RangeFiles(handle func(astInfo *AstFileInfo, file *ast.File) error) error {
	for file, info := range pkgs.files {
		dir := strings.TrimPrefix(filepath.Dir(info.Path), pkgs.workdir+"/")
		if _, ok := pkgs.excludes[dir]; ok {
			continue
		}

		if strings.HasSuffix(file.Name.String(), "_test.go") {
			continue
		}

		if _, ok := pkgs.excludes[info.Path]; ok {
			continue
		}

		if err := handle(info, file); err != nil {
			return err
		}
	}
	return nil
}

func (pkgs *PackagesDefinitions) CollectAstFile(packageDir, path string, astFile *ast.File, fset *token.FileSet) {
	if pkgs.files == nil {
		pkgs.files = make(map[*ast.File]*AstFileInfo)
	}

	pkgs.files[astFile] = &AstFileInfo{
		File:        astFile,
		Path:        path,
		FileSet:     fset,
		PackagePath: packageDir,
	}

	if len(packageDir) == 0 {
		return
	}

	if pkgs.packages == nil {
		pkgs.packages = make(map[string]*PackageDefinitions)
	}

	if pd, ok := pkgs.packages[packageDir]; ok {
		pd.Files[path] = astFile
	} else {
		pkgs.packages[packageDir] = &PackageDefinitions{
			Name:  astFile.Name.Name,
			Files: map[string]*ast.File{path: astFile},
		}
	}
}
