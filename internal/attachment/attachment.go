package attachment

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	MaxFiles = 10
	// MaxSize is the largest file the bot uploads. Slack accepts more, but
	// slide previews, decks and data exports stay well below this.
	MaxSize = 100 << 20
)

// An allowlist, not a denylist: source files or .env must not leave even from
// inside a work area.
var allowedExtensions = map[string]struct{}{
	".pdf":  {},
	".png":  {},
	".jpg":  {},
	".jpeg": {},
	".gif":  {},
	".webp": {},
	".pptx": {},
	".csv":  {},
}

type File struct {
	Path string
	info fs.FileInfo
}

func (f File) Name() string { return filepath.Base(f.Path) }

func (f File) Size() int64 { return f.info.Size() }

type area struct{ lexical, real string }

// Reason is a code, not text, so the bot can word it in the operator's
// language.
type Reason int

const (
	TooMany Reason = iota + 1
	NotAllowedType
	NotFound
	Unreadable
	OutsideArea
	NotRegular
	HardLinked
	Empty
	TooLarge
	UploadFailed
)

type Rejection struct {
	Path   string
	Reason Reason
}

func Resolve(roots []string, cwd string, paths []string) ([]File, []Rejection) {
	var areas []area
	for _, root := range roots {
		if root == "" || !filepath.IsAbs(root) {
			continue
		}
		real, err := filepath.EvalSymlinks(root)
		if err != nil {
			continue
		}
		areas = append(areas, area{lexical: filepath.Clean(root), real: real})
	}

	var files []File
	var rejections []Rejection
	seen := make(map[string]struct{})
	for _, path := range paths {
		if !filepath.IsAbs(path) {
			path = filepath.Join(cwd, path)
		}
		path = filepath.Clean(path)
		file, reason := resolveOne(path, areas)
		if reason != 0 {
			rejections = append(rejections, Rejection{Path: path, Reason: reason})
			continue
		}
		// The real path catches one file listed through different paths.
		if _, ok := seen[file.Path]; ok {
			continue
		}
		seen[file.Path] = struct{}{}
		if len(files) >= MaxFiles {
			rejections = append(rejections, Rejection{Path: path, Reason: TooMany})
			continue
		}
		files = append(files, file)
	}
	return files, rejections
}

func resolveOne(path string, areas []area) (File, Reason) {
	ext := strings.ToLower(filepath.Ext(path))
	if _, ok := allowedExtensions[ext]; !ok {
		return File{}, NotAllowedType
	}
	real, err := filepath.EvalSymlinks(path)
	if errors.Is(err, fs.ErrNotExist) {
		return File{}, NotFound
	} else if err != nil {
		return File{}, Unreadable
	}
	if !inAreas(areas, path, real) {
		return File{}, OutsideArea
	}
	info, err := os.Lstat(real)
	if err != nil {
		return File{}, Unreadable
	}
	if !info.Mode().IsRegular() {
		return File{}, NotRegular
	}
	// A hard link shares the inode of a file that may live outside the work
	// area, such as a credential, and no path check can tell them apart.
	if linkCount(info) > 1 {
		return File{}, HardLinked
	}
	if info.Size() == 0 {
		return File{}, Empty
	}
	if info.Size() > MaxSize {
		return File{}, TooLarge
	}
	return File{Path: real, info: info}, 0
}

// The real path must match the lexical one, so no symbolic link inside the work
// area redirects the file.
func inAreas(areas []area, path, real string) bool {
	for _, area := range areas {
		lexicalRel, ok := within(area.lexical, path)
		if !ok {
			continue
		}
		if realRel, ok := within(area.real, real); ok && realRel == lexicalRel {
			return true
		}
	}
	return false
}

// Checked again so a file replaced after Resolve is not uploaded.
func (f File) Open() (*os.File, error) {
	file, err := os.Open(f.Path)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}
	if !os.SameFile(info, f.info) || !info.Mode().IsRegular() ||
		info.Size() != f.info.Size() || !info.ModTime().Equal(f.info.ModTime()) {
		file.Close()
		return nil, errors.New("the file changed after it was checked")
	}
	return file, nil
}

func within(root, path string) (string, bool) {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", false
	}
	return rel, true
}
