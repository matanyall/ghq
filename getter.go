package main

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/x-motemen/ghq/logger"
)

var seen sync.Map

func getRepoLock(localRepoRoot string) bool {
	_, loaded := seen.LoadOrStore(localRepoRoot, struct{}{})
	return !loaded
}

type getInfo struct {
	localRepository *LocalRepository
}

type getter struct {
	update, shallow, silent, ssh, recursive, bare bool
	vcs, branch, partial                          string
}

func (g *getter) get(argURL string) (getInfo, error) {
	u, err := newURL(argURL, g.ssh, false)
	if err != nil {
		return getInfo{}, fmt.Errorf("could not parse URL %q: %w", argURL, err)
	}
	branch := g.branch
	if pos := strings.LastIndexByte(u.Path, '@'); pos >= 0 {
		u.Path, branch = u.Path[:pos], u.Path[pos+1:]
	}
	remote, err := NewRemoteRepository(u)
	if err != nil {
		return getInfo{}, err
	}

	return g.getRemoteRepository(remote, branch)
}

// getRemoteRepository clones or updates a remote repository remote.
// If doUpdate is true, updates the locally cloned repository. Otherwise does nothing.
// If isShallow is true, does shallow cloning. (no effect if already cloned or the VCS is Mercurial and git-svn)
func (g *getter) getRemoteRepository(remote RemoteRepository, branch string) (getInfo, error) {
	remoteURL := remote.URL()

	// Determine if the special "worktree mode" feature is enabled. This feature
	// will clone the repository under an additional sub-directory that matches
	// the branch name (e.g. org/repo/main). When no branch is specified by the
	// user, we will try to discover the default branch of the remote (Git only)
	// so that the directory name still reflects an actual branch. If detection
	// fails we simply fall back to "main".
	worktreeMode := isWorktreeModeEnabled()
	var branchForDir = branch
	if worktreeMode && branchForDir == "" {
		// Attempt to auto-detect default branch for Git remotes. We ignore any
		// error and use the detected value when available.
		branchForDir = detectDefaultGitBranch(remoteURL.String())
		if branchForDir == "" {
			branchForDir = "main"
		}
	}

	local, err := LocalRepositoryFromURL(remoteURL, g.bare)
	if err != nil {
		return getInfo{}, err
	}
	info := getInfo{
		localRepository: local,
	}

	var (
		fpath   = local.FullPath
		newPath = false
	)

	// If worktree mode is enabled, append the branch directory to where we are
	// going to place the repository.
	if worktreeMode {
		fpath = filepath.Join(fpath, branchForDir)
		// Refresh LocalRepository information so that other parts of ghq (e.g.
		// look, list) are aware of the actual location.
		if lr, err := LocalRepositoryFromFullPath(fpath, nil); err == nil {
			info.localRepository = lr
			local = lr
		}
	}

	_, err = os.Stat(fpath)
	if err != nil {
		if os.IsNotExist(err) {
			newPath = true
			err = nil
		}
		if err != nil {
			return getInfo{}, err
		}
	}

	switch {
	case newPath:
		if remoteURL.Scheme == "codecommit" {
			logger.Log("clone", fmt.Sprintf("%s -> %s", remoteURL.Opaque, fpath))
		} else {
			logger.Log("clone", fmt.Sprintf("%s -> %s", remoteURL, fpath))
		}
		var (
			localRepoRoot = fpath
			repoURL       = remoteURL
		)
		vcs, ok := vcsRegistry[g.vcs]
		if !ok {
			vcs, repoURL, err = remote.VCS()
			if err != nil {
				return getInfo{}, err
			}
		}
		if l := detectLocalRepoRoot(remoteURL.Path, repoURL.Path); l != "" {
			localRepoRoot = filepath.Join(local.RootPath, remoteURL.Hostname(), l)
		}

		// In worktree mode, append branch directory here as well (this path is
		// ultimately provided to the VCS backend).
		if worktreeMode {
			localRepoRoot = filepath.Join(localRepoRoot, branchForDir)
		}

		if worktreeMode {
			// Ensure parent directory for branch directory exists (org/repo)
			if err := os.MkdirAll(filepath.Dir(localRepoRoot), 0o755); err != nil {
				return getInfo{}, err
			}
		}

		if g.bare {
			localRepoRoot = localRepoRoot + ".git"
		}

		if remoteURL.Scheme == "codecommit" {
			repoURL, _ = url.Parse(remoteURL.Opaque)
		}
		if getRepoLock(localRepoRoot) {
			return info,
				vcs.Clone(&vcsGetOption{
					url:       repoURL,
					dir:       localRepoRoot,
					shallow:   g.shallow,
					silent:    g.silent,
					branch:    branchForDir,
					recursive: g.recursive,
					bare:      g.bare,
					partial:   g.partial,
				})
		}
		return info, nil
	case g.update:
		logger.Log("update", fpath)
		vcs, localRepoRoot := local.VCS()
		if vcs == nil {
			return getInfo{}, fmt.Errorf("failed to detect VCS for %q", fpath)
		}
		repoURL := remoteURL
		if remoteURL.Scheme == "codecommit" {
			repoURL, _ = url.Parse(remoteURL.Opaque)
		}
		if getRepoLock(localRepoRoot) {
			return info, vcs.Update(&vcsGetOption{
				url:       repoURL,
				dir:       localRepoRoot,
				silent:    g.silent,
				recursive: g.recursive,
				bare:      g.bare,
			})
		}
		return info, nil
	}
	logger.Log("exists", fpath)
	return info, nil
}

func detectLocalRepoRoot(remotePath, repoPath string) string {
	remotePath = strings.TrimSuffix(strings.TrimSuffix(remotePath, "/"), ".git")
	repoPath = strings.TrimSuffix(strings.TrimSuffix(repoPath, "/"), ".git")
	pathParts := strings.Split(repoPath, "/")
	pathParts = pathParts[1:]
	for i := 0; i < len(pathParts); i++ {
		subPath := "/" + path.Join(pathParts[i:]...)
		if subIdx := strings.Index(remotePath, subPath); subIdx >= 0 {
			return remotePath[0:subIdx] + subPath
		}
	}
	return ""
}
