package main

import (
    "path/filepath"
    "testing"

    "github.com/Songmu/gitconfig"
)

func TestGet_WorktreeMode_EnvVar(t *testing.T) {
    withFakeGitBackend(t, func(t *testing.T, tmpRoot string, cloneArgs *_cloneArgs, _ *_updateArgs) {
        setEnv(t, envGhqWorktreeMode, "1")

        app := newApp()
        branchName := "feature-xyz"
        if err := app.Run([]string{"", "get", "-branch", branchName, "motemen/ghq-test-repo"}); err != nil {
            t.Fatalf("app.Run(): %s", err)
        }

        expectDir := filepath.Join(tmpRoot, "github.com", "motemen", "ghq-test-repo", branchName)
        if filepath.ToSlash(cloneArgs.local) != filepath.ToSlash(expectDir) {
            t.Errorf("cloneArgs.local: got %s, want %s", filepath.ToSlash(cloneArgs.local), filepath.ToSlash(expectDir))
        }
        if cloneArgs.branch != branchName {
            t.Errorf("cloneArgs.branch: got %s, want %s", cloneArgs.branch, branchName)
        }
    })
}

func TestGet_WorktreeMode_GitConfig(t *testing.T) {
    withFakeGitBackend(t, func(t *testing.T, tmpRoot string, cloneArgs *_cloneArgs, _ *_updateArgs) {
        t.Cleanup(gitconfig.WithConfig(t, `
[ghq]
  worktreeMode = true
`))

        app := newApp()
        branchName := "dev"
        if err := app.Run([]string{"", "get", "-branch", branchName, "motemen/ghq-test-repo"}); err != nil {
            t.Fatalf("app.Run(): %s", err)
        }

        expectDir := filepath.Join(tmpRoot, "github.com", "motemen", "ghq-test-repo", branchName)
        if filepath.ToSlash(cloneArgs.local) != filepath.ToSlash(expectDir) {
            t.Errorf("cloneArgs.local: got %s, want %s", filepath.ToSlash(cloneArgs.local), filepath.ToSlash(expectDir))
        }
    })
}