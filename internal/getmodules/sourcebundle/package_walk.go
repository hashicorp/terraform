package sourcebundle

import (
	"fmt"
	"os"
	"path/filepath"
)

func packagePrepareWalkFn(root string, ignoreRules []rule) filepath.WalkFunc {
	return func(absPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get the relative path from the current src directory.
		relPath, err := filepath.Rel(root, absPath)
		if err != nil {
			return fmt.Errorf("failed to get relative path for file %q: %w", absPath, err)
		}
		if relPath == "." {
			return nil
		}

		if matchIgnoreRule(relPath, ignoreRules) {
			err := os.RemoveAll(absPath)
			if err != nil {
				return fmt.Errorf("failed to remove ignored file %s: %s", relPath, err)
			}
			return nil
		}

		// For directories we also need to check with a path separator on the
		// end, which ignores entire subtrees.
		if info.IsDir() {
			if m := matchIgnoreRule(relPath+string(os.PathSeparator), ignoreRules); m {
				err := os.RemoveAll(absPath)
				if err != nil {
					return fmt.Errorf("failed to remove ignored file %s: %s", relPath, err)
				}
				return nil
			}
		}

		// If we get here then we have a file or directory that isn't
		// covered by the ignore rules, but we still need to make sure it's
		// valid for inclusion in a source bundle.
		// We only allow regular files, directories, and symlinks to either
		// of those as long as they are under the root directory prefix.
		absRoot, err := filepath.Abs(root)
		if err != nil {
			return fmt.Errorf("failed to get absolute path for root directory %q: %w", root, err)
		}
		absRoot, err = filepath.EvalSymlinks(absRoot)
		if err != nil {
			return fmt.Errorf("failed to get absolute path for root directory %q: %w", root, err)
		}
		reAbsPath := filepath.Join(absRoot, relPath)
		realPath, err := filepath.EvalSymlinks(reAbsPath)
		if err != nil {
			return fmt.Errorf("failed to get real path for sub-path %q: %w", relPath, err)
		}
		realPathRel, err := filepath.Rel(absRoot, realPath)
		if err != nil {
			return fmt.Errorf("failed to get real relative path for sub-path %q: %w", relPath, err)
		}

		// After all of the above we can finally safely test whether the
		// transformed path is "local", meaning that it only descends down
		// from the real root.
		if !filepath.IsLocal(realPathRel) {
			return fmt.Errorf("module package path %q is symlink traversing out of the package root", relPath)
		}

		// The real referent must also be either a regular file or a directory.
		// (Not, for example, a Unix device node or socket or other such oddities.)
		lInfo, err := os.Lstat(realPath)
		if err != nil {
			return fmt.Errorf("failed to stat %q: %w", realPath, err)
		}
		if !(lInfo.Mode().IsRegular() || lInfo.Mode().IsDir()) {
			return fmt.Errorf("module package path %q is not a regular file or directory", relPath)
		}

		return nil
	}
}
