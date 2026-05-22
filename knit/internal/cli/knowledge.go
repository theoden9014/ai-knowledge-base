package cli

import (
	"fmt"
	"path/filepath"
)

// knowledgeResolver is responsible for resolving the absolute path to
// the auto-detected knowledge/ directory from the Runtime.
//
// Resolution order:
//  1. Search upward on Runtime.Fsys from Runtime.Getwd for a "knowledge" directory
//     (sharing the same findUpwards path as scopeResolver).
//  2. If nothing is found, return ErrKnowledgeDirNotFound.
//
// scopeResolver and knowledgeResolver use similar search strategies, but
// they are separated because their purposes differ: the former resolves
// the project's base, while the latter resolves the source pack
// location. The only shared low-level pieces are
// scopeResolver.findUpwards and Runtime.Fsys.
//
// This type is an internal implementation detail and is not exported.
type knowledgeResolver struct {
	rt *Runtime
}

// newKnowledgeResolver constructs a knowledgeResolver from a Runtime.
func newKnowledgeResolver(rt *Runtime) *knowledgeResolver {
	return &knowledgeResolver{rt: rt}
}

// knowledgeDirName is the directory name targeted by auto-detection.
const knowledgeDirName = "knowledge"

// resolve returns the absolute path to auto-detected knowledge/. If nothing is
// found, it returns ErrKnowledgeDirNotFound.
func (r *knowledgeResolver) resolve() (string, error) {
	// auto-detect by walking upwards for a "knowledge" directory.
	wd, err := r.rt.Getwd()
	if err != nil {
		return "", fmt.Errorf("cli: getwd: %w", err)
	}
	sr := newScopeResolver(r.rt)
	found, ok, err := sr.findUpwards(FsPathFromAbs(wd), []string{knowledgeDirName})
	if err != nil {
		return "", err
	}
	if !ok {
		return "", ErrKnowledgeDirNotFound
	}
	return filepath.Join(found.Abs(), knowledgeDirName), nil
}
