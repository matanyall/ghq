package main

import (
    "path/filepath"
    "testing"

    "github.com/Songmu/gitconfig"
    "os"
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

func TestGet_WorktreeMode_DefaultBranchAutoDetection(t *testing.T) {
    withFakeGitBackend(t, func(t *testing.T, tmpRoot string, cloneArgs *_cloneArgs, _ *_updateArgs) {
        setEnv(t, envGhqWorktreeMode, "1")

        // Stub git executable in PATH
        tmpBin, err := os.MkdirTemp("", "ghq-test-bin")
        if err != nil { t.Fatalf("MkdirTemp: %v", err) }
        script := "#!/bin/sh\nif [ \"$1\" = \"ls-remote\" ]; then echo 'ref: refs/heads/develop\tHEAD'; else echo 'git: unexpected call' >&2; exit 1; fi\n"
        gitPath := filepath.Join(tmpBin, "git")
        if err := os.WriteFile(gitPath, []byte(script), 0o755); err != nil {
            t.Fatalf("WriteFile: %v", err)
        }
        origPath := os.Getenv("PATH")
        setEnv(t, "PATH", tmpBin+string(os.PathListSeparator)+origPath)

        app := newApp()
        if err := app.Run([]string{"", "get", "motemen/ghq-test-repo"}); err != nil {
            t.Fatalf("app.Run(): %s", err)
        }

        expectDir := filepath.Join(tmpRoot, "github.com", "motemen", "ghq-test-repo", "develop")
        if filepath.ToSlash(cloneArgs.local) != filepath.ToSlash(expectDir) {
            t.Errorf("cloneArgs.local: got %s, want %s", cloneArgs.local, expectDir)
        }
        if cloneArgs.branch != "develop" {
            t.Errorf("cloneArgs.branch: got %s, want develop", cloneArgs.branch)
        }
        if _, err := os.Stat(expectDir); err != nil {
            t.Errorf("expectDir should exist: %v", err)
        }
    })
}

func TestGet_WorktreeMode_DefaultsToMainWhenDetectionFails(t *testing.T) {
    withFakeGitBackend(t, func(t *testing.T, tmpRoot string, cloneArgs *_cloneArgs, _ *_updateArgs) {
        setEnv(t, envGhqWorktreeMode, "1")
        // override PATH with dummy git command that exits error
        tmpBin, _ := os.MkdirTemp("", "ghq-test-bin")
        script := "#!/bin/sh\nexit 1\n"
        os.WriteFile(filepath.Join(tmpBin, "git"), []byte(script), 0o755)
        origPath := os.Getenv("PATH")
        setEnv(t, "PATH", tmpBin+string(os.PathListSeparator)+origPath)

        app := newApp()
        if err := app.Run([]string{"", "get", "motemen/ghq-test-repo"}); err != nil {
            t.Fatalf("app.Run(): %s", err)
        }
        expectDir := filepath.Join(tmpRoot, "github.com", "motemen", "ghq-test-repo", "main")
        if filepath.ToSlash(cloneArgs.local) != filepath.ToSlash(expectDir) {
            t.Errorf("cloneArgs.local: got %s, want %s", cloneArgs.local, expectDir)
        }
    })
}

func TestGet_DisabledWorktreeMode(t *testing.T) {
    withFakeGitBackend(t, func(t *testing.T, tmpRoot string, cloneArgs *_cloneArgs, _ *_updateArgs) {
        // Ensure feature disabled
        setEnv(t, envGhqWorktreeMode, "")

        branch := "hello"
        app := newApp()
        if err := app.Run([]string{"", "get", "-branch", branch, "motemen/ghq-test-repo"}); err != nil {
            t.Fatalf("app.Run(): %s", err)
        }
        expectDir := filepath.Join(tmpRoot, "github.com", "motemen", "ghq-test-repo")
        if filepath.ToSlash(cloneArgs.local) != filepath.ToSlash(expectDir) {
            t.Errorf("cloneArgs.local: got %s, want %s", cloneArgs.local, expectDir)
        }
    })
}

func TestGet_WorktreeMode_BareRepo(t *testing.T) {
    withFakeGitBackend(t, func(t *testing.T, tmpRoot string, cloneArgs *_cloneArgs, _ *_updateArgs) {
        setEnv(t, envGhqWorktreeMode, "1")
        branch := "feat"
        app := newApp()
        if err := app.Run([]string{"", "get", "--bare", "-branch", branch, "motemen/ghq-test-repo"}); err != nil {
            t.Fatalf("app.Run(): %s", err)
        }
        expectDir := filepath.Join(tmpRoot, "github.com", "motemen", "ghq-test-repo", branch+".git")
        if filepath.ToSlash(cloneArgs.local) != filepath.ToSlash(expectDir) {
            t.Errorf("cloneArgs.local: got %s, want %s", cloneArgs.local, expectDir)
        }
    })
}

func TestDetectDefaultGitBranch(t *testing.T) {
    // Successful detection
    tmpBin, _ := os.MkdirTemp("", "ghq-test-bin")
    script := "#!/bin/sh\nif [ \"$1\" = \"ls-remote\" ]; then echo 'ref: refs/heads/main\tHEAD'; else exit 1; fi\n"
    gitPath := filepath.Join(tmpBin, "git")
    os.WriteFile(gitPath, []byte(script), 0o755)
    origPath := os.Getenv("PATH")
    os.Setenv("PATH", tmpBin+string(os.PathListSeparator)+origPath)
    defer os.Setenv("PATH", origPath)
    defer os.RemoveAll(tmpBin)

    if got := detectDefaultGitBranch("https://example.com/repo.git"); got != "main" {
        t.Errorf("detectDefaultGitBranch success: got %s want main", got)
    }

    // Failing detection (exit 1) returns empty string
    tmpBin2, _ := os.MkdirTemp("", "ghq-test-bin")
    script2 := "#!/bin/sh\nexit 1\n"
    os.WriteFile(filepath.Join(tmpBin2, "git"), []byte(script2), 0o755)
    os.Setenv("PATH", tmpBin2+string(os.PathListSeparator)+origPath)
    defer os.RemoveAll(tmpBin2)

    if got := detectDefaultGitBranch("https://example.com/repo.git"); got != "" {
        t.Errorf("detectDefaultGitBranch failure: got %s want empty", got)
    }
}