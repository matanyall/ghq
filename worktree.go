// worktree.go

package main

import (
    "bufio"
    "bytes"
    "os"
    "os/exec"
    "strings"

    "github.com/Songmu/gitconfig"
)

const envGhqWorktreeMode = "GHQ_WORKTREE_MODE"

// isWorktreeModeEnabled returns true if the worktree mode feature is enabled
// either by environment variable GHQ_WORKTREE_MODE or git configuration key
// ghq.worktreeMode (boolean) or ghq.worktree-mode (boolean).
func isWorktreeModeEnabled() bool {
    // Environment variable takes highest priority. Any value other than an empty
    // string or explicit "0"/"false" (case-insensitive) enables the feature.
    if v, ok := os.LookupEnv(envGhqWorktreeMode); ok {
        v = strings.ToLower(strings.TrimSpace(v))
        if v == "" || v == "0" || v == "false" {
            return false
        }
        return true
    }

    // Check git config keys. If either is set to true, enable the mode. When the
    // key exists but cannot be parsed as a boolean, gitconfig.Bool returns an
    // error; we purposely ignore that and treat it as not-set.
    if b, err := gitconfig.Bool("ghq.worktreeMode"); err == nil {
        return b
    }
    if b, err := gitconfig.Bool("ghq.worktree-mode"); err == nil {
        return b
    }
    return false
}

// detectDefaultGitBranch tries to discover the default branch name of a remote
// Git repository. It shells out to `git ls-remote --symref <url> HEAD` and
// parses the symbolic reference target. If detection fails, an empty string is
// returned.
func detectDefaultGitBranch(remoteURL string) string {
    cmd := exec.Command("git", "ls-remote", "--symref", remoteURL, "HEAD")
    var stdout bytes.Buffer
    cmd.Stdout = &stdout
    if err := cmd.Run(); err != nil {
        return ""
    }
    scanner := bufio.NewScanner(&stdout)
    for scanner.Scan() {
        line := scanner.Text()
        // Expected line format: "ref: refs/heads/<branch>\tHEAD"
        if !strings.HasSuffix(line, "\tHEAD") {
            continue
        }
        fields := strings.SplitN(line, "\t", 2)
        if len(fields) < 1 {
            continue
        }
        refPart := strings.TrimSpace(fields[0])
        const prefix = "ref: refs/heads/"
        if strings.HasPrefix(refPart, prefix) {
            return strings.TrimPrefix(refPart, prefix)
        }
    }
    return ""
}