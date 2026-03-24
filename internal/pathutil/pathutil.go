package pathutil

import (
	"errors"
	"path"
	"strings"
	"unicode"
)

// MaxDepth is the maximum number of folder levels allowed (e.g. a/b/c/file = depth 3).
const MaxDepth = 3

var (
	ErrEmptyPath     = errors.New("path must not be empty")
	ErrInvalidChars  = errors.New("path contains invalid characters; only a-z A-Z 0-9 space . _ - are allowed in segment names")
	ErrTooDeep       = errors.New("path exceeds maximum folder depth of 3")
	ErrTrailingSlash = errors.New("path must not end with a slash")
	ErrEmptySegment  = errors.New("path must not contain empty segments (double slashes)")
)

// Validate checks that p is a valid canonical file path.
//
// Rules:
//   - Non-empty
//   - No trailing slash
//   - No empty segments (double slashes)
//   - Each segment contains only [a-zA-Z0-9 ._-]
//   - At most MaxDepth folder levels (segments before the last one)
func Validate(p string) error {
	if p == "" {
		return ErrEmptyPath
	}
	if strings.HasSuffix(p, "/") {
		return ErrTrailingSlash
	}
	segments := strings.Split(p, "/")
	for _, seg := range segments {
		if seg == "" {
			return ErrEmptySegment
		}
		if !isASCIISafe(seg) {
			return ErrInvalidChars
		}
	}
	// depth = number of folder segments (all except the last, which is the filename)
	depth := len(segments) - 1
	if depth > MaxDepth {
		return ErrTooDeep
	}
	return nil
}

// isASCIISafe returns true if every rune in s is in [a-zA-Z0-9 ._-].
func isASCIISafe(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			continue
		}
		switch r {
		case ' ', '.', '_', '-':
			continue
		default:
			return false
		}
	}
	return true
}

// Normalize cleans redundant slashes and applies path.Clean, returning the
// cleaned result. It does NOT add the Main/ prefix — that is the caller's job.
func Normalize(p string) string {
	return path.Clean(p)
}

// Basename returns the last path element (the filename without any folder prefix).
func Basename(p string) string {
	return path.Base(p)
}

// Dir returns the directory portion of the path (everything except the basename).
// Returns "." if the path has no directory.
func Dir(p string) string {
	return path.Dir(p)
}

// WithMainPrefix ensures p begins with "Main/". If p already contains a slash
// it is assumed to already carry a folder prefix and is returned unchanged.
func WithMainPrefix(p string) string {
	if strings.Contains(p, "/") {
		return p
	}
	return "Main/" + p
}

// FolderPrefix returns the prefix string (with trailing slash) that must be
// used when listing all objects inside folder f.
// E.g. FolderPrefix("Main") == "Main/".
func FolderPrefix(folder string) string {
	folder = strings.TrimRight(folder, "/")
	if folder == "" {
		return ""
	}
	return folder + "/"
}

// ImmediateChildren returns the unique direct-child folder names visible under
// parentPrefix from a list of canonical file paths.
//
// parentPrefix must end with "/" (or be empty string for root-level folders).
// The returned names do NOT include a trailing slash.
func ImmediateChildren(parentPrefix string, paths []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, p := range paths {
		if !strings.HasPrefix(p, parentPrefix) {
			continue
		}
		rest := strings.TrimPrefix(p, parentPrefix)
		// rest must contain at least one more "/" to represent a sub-folder
		idx := strings.Index(rest, "/")
		if idx < 0 {
			// direct file, not a subfolder
			continue
		}
		folderName := rest[:idx]
		if _, exists := seen[folderName]; !exists {
			seen[folderName] = struct{}{}
			out = append(out, folderName)
		}
	}
	return out
}
