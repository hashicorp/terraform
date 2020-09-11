package complete

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// PredictDirs will search for directories in the given started to be typed
// path, if no path was started to be typed, it will complete to directories
// in the current working directory.
func PredictDirs(pattern string) Predictor {
	return files(pattern, false)
}

// PredictFiles will search for files matching the given pattern in the started to
// be typed path, if no path was started to be typed, it will complete to files that
// match the pattern in the current working directory.
// To match any file, use "*" as pattern. To match go files use "*.go", and so on.
func PredictFiles(pattern string) Predictor {
	return files(pattern, true)
}

func files(pattern string, allowFiles bool) PredictFunc {

	// search for files according to arguments,
	// if only one directory has matched the result, search recursively into
	// this directory to give more results.
	return func(a Args) (prediction []string) {
		prediction = predictFiles(a, pattern, allowFiles)

		// if the number of prediction is not 1, we either have many results or
		// have no results, so we return it.
		if len(prediction) != 1 {
			return
		}

		// only try deeper, if the one item is a directory
		if stat, err := os.Stat(prediction[0]); err != nil || !stat.IsDir() {
			return
		}

		a.Last = prediction[0]
		return predictFiles(a, pattern, allowFiles)
	}
}

func predictFiles(a Args, pattern string, allowFiles bool) []string {
	if strings.HasSuffix(a.Last, "/..") {
		return nil
	}

	dir := directory(a.Last)
	files := listFiles(dir, pattern, allowFiles)

	// add dir if match
	files = append(files, dir)

	return PredictFilesSet(files).Predict(a)
}

// directory gives the directory of the given partial path
// in case that it is not, we fall back to the current directory.
func directory(path string) string {
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return fixPathForm(path, path)
	}
	dir := filepath.Dir(path)
	if info, err := os.Stat(dir); err == nil && info.IsDir() {
		return fixPathForm(path, dir)
	}
	return "./"
}

// PredictFilesSet predict according to file rules to a given set of file names
func PredictFilesSet(files []string) PredictFunc {
	return func(a Args) (prediction []string) {
		// add all matching files to prediction
		for _, f := range files {
			f = fixPathForm(a.Last, f)

			// test matching of file to the argument
			if matchFile(f, a.Last) {
				prediction = append(prediction, f)
			}
		}
		return
	}
}

func listFiles(dir, pattern string, allowFiles bool) []string {
	// set of all file names
	m := map[string]bool{}

	// list files
	if files, err := filepath.Glob(filepath.Join(dir, pattern)); err == nil {
		for _, f := range files {
			if stat, err := os.Stat(f); err != nil || stat.IsDir() || allowFiles {
				m[f] = true
			}
		}
	}

	// list directories
	if dirs, err := ioutil.ReadDir(dir); err == nil {
		for _, d := range dirs {
			if d.IsDir() {
				m[filepath.Join(dir, d.Name())] = true
			}
		}
	}

	list := make([]string, 0, len(m))
	for k := range m {
		list = append(list, k)
	}
	return list
}

// MatchFile returns true if prefix can match the file
func matchFile(file, prefix string) bool {
	// special case for current directory completion
	if file == "./" && (prefix == "." || prefix == "") {
		return true
	}
	if prefix == "." && strings.HasPrefix(file, ".") {
		return true
	}

	file = strings.TrimPrefix(file, "./")
	prefix = strings.TrimPrefix(prefix, "./")

	return strings.HasPrefix(file, prefix)
}

// fixPathForm changes a file name to a relative name
func fixPathForm(last string, file string) string {
	// get wording directory for relative name
	workDir, err := os.Getwd()
	if err != nil {
		return file
	}

	abs, err := filepath.Abs(file)
	if err != nil {
		return file
	}

	// if last is absolute, return path as absolute
	if filepath.IsAbs(last) {
		return fixDirPath(abs)
	}

	rel, err := filepath.Rel(workDir, abs)
	if err != nil {
		return file
	}

	// fix ./ prefix of path
	if rel != "." && strings.HasPrefix(last, ".") {
		rel = "./" + rel
	}

	return fixDirPath(rel)
}

func fixDirPath(path string) string {
	info, err := os.Stat(path)
	if err == nil && info.IsDir() && !strings.HasSuffix(path, "/") {
		path += "/"
	}
	return path
}
