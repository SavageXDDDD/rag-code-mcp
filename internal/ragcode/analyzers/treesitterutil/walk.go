package treesitterutil

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// FileVisitor is invoked once per file picked up by WalkSourceFiles. Return
// a non-nil error to abort the walk.
type FileVisitor func(path string, content []byte) error

// WalkSourceFiles invokes visit for every regular file under any of roots
// whose extension is listed in exts. Extensions must include the leading
// dot, e.g. ".java" — they're compared against filepath.Ext, which always
// returns that form.
//
// Pruning rules:
//   - A directory is skipped when its own base name appears in skipDirs.
//     Parent path segments are not considered, so a project located at
//     /home/me/build/myrepo is still walked even though "build" is a
//     conventional skip-dir.
//   - When skipTests is true, files whose base name (extension stripped,
//     lower-cased) starts with "test_", ends with "_test", or ends with
//     "test" are skipped. That covers the three common conventions:
//     Python (test_foo.py), Go (foo_test.go), Java (FooTest.java). Note
//     that "tester.go" / "latest.java" are false-positives — by design,
//     accept the rare collision.
//
// roots may contain a mix of files and directories. A directory root is
// walked recursively; a regular-file root is still subject to the
// extension filter (pass a matching extension if you want it indexed).
//
// Errors:
//   - Failures from os.Stat on a root entry are swallowed (the entry is
//     simply skipped) — passing a non-existent path is not fatal.
//   - Failures from os.ReadDir abort the walk and propagate.
//   - Failures from os.ReadFile on a matched file are swallowed (the file
//     is skipped); a non-nil error returned from visit aborts the walk
//     and propagates as-is.
//
// The walk is single-threaded so a language analyzer's parser usage stays
// simple. Parallelise at the caller layer if needed — ParserPool is safe
// for concurrent Get/Put.
func WalkSourceFiles(roots, exts, skipDirs []string, skipTests bool, visit FileVisitor) error {
	for _, root := range roots {
		fi, err := os.Stat(root)
		if err != nil {
			continue
		}
		if fi.IsDir() {

			if slices.Contains(skipDirs, fi.Name()) || slices.Contains(skipDirs, fi.Name()) {
				continue
			}

			dirEnts, err := os.ReadDir(root)
			if err != nil {
				return err
			}

			newRoots := make([]string, len(dirEnts))

			for i, dirEnt := range dirEnts {
				newRoots[i] = filepath.Join(root, dirEnt.Name())
			}

			err = WalkSourceFiles(newRoots, exts, skipDirs, skipTests, visit)
			if err != nil {
				return err
			}
		} else {
			extension := filepath.Ext(fi.Name())
			if slices.Contains(exts, extension) {
				if skipTests {
					base := strings.TrimSuffix(strings.ToLower(fi.Name()), extension)
					shouldSkip := strings.HasPrefix(base, "test_") ||
						strings.HasSuffix(base, "_test") ||
						strings.HasSuffix(base, "test")

					if shouldSkip {
						continue
					}
				}

				var contents []byte
				contents, err = os.ReadFile(root)
				if err != nil {
					continue
				}
				err = visit(root, contents)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
