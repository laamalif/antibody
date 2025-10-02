package project

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

var replaces = []struct{ a, b string }{
	{":", "-COLON-"},
	{"/", "-SLASH-"},
	{"@", "-AT-"},
}

// FromURL converts the given URL to a folder name
func FromURL(url string) string {
	result := url
	for _, replace := range replaces {
		result = strings.Replace(result, replace.a, replace.b, -1)
	}
	return result
}

// ToURL converts the given folder to an URL
func ToURL(folder string) string {
	result := folder
	for _, replace := range replaces {
		result = strings.Replace(result, replace.b, replace.a, -1)
	}
	return result
}

// nolint: gochecknoglobals
var gitCmdEnv = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_ASKPASS=0", "SSH_ASKPASS=0")

type gitProject struct {
	URL     string
	Version string
	folder  string
	inner   string
}

// NewClonedGit is a git project that was already cloned, so, only Update
// will work here.
func NewClonedGit(home, folderName string) Project {
	folderPath := filepath.Join(home, folderName)
	version, err := branch(folderPath)
	if err != nil {
		version = ""
	}
	url := ToURL(folderName)
	return gitProject{
		folder:  folderPath,
		Version: version,
		URL:     url,
	}
}

const (
	branchMarker = "branch:"
	pathMarker   = "path:"
)

// NewGit A git project can be any repository in any given branch. It will
// be downloaded to the provided cwd
func NewGit(cwd, line string) Project {
	version := ""
	inner := ""
	parts := strings.Split(line, " ")
	for _, part := range parts {
		if strings.HasPrefix(part, branchMarker) {
			version = strings.ReplaceAll(part, branchMarker, "")
		}
		if strings.HasPrefix(part, pathMarker) {
			inner = strings.ReplaceAll(part, pathMarker, "")
		}
	}
	repo := parts[0]
	url := "https://github.com/" + repo
	switch {
	case strings.HasPrefix(repo, "http://"):
		fallthrough
	case strings.HasPrefix(repo, "https://"):
		fallthrough
	case strings.HasPrefix(repo, "git://"):
		fallthrough
	case strings.HasPrefix(repo, "ssh://"):
		fallthrough
	case strings.HasPrefix(repo, "git@gitlab.com:"):
		fallthrough
	case strings.HasPrefix(repo, "git@github.com:"):
		url = repo
	}
	folder := filepath.Join(cwd, FromURL(url))
	return gitProject{
		Version: version,
		URL:     url,
		folder:  folder,
		inner:   inner,
	}
}

// nolint: gochecknoglobals
var locks sync.Map

func (g gitProject) Download() error {
	l, _ := locks.LoadOrStore(g.folder, &sync.Mutex{})
	lock := l.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()
	if _, err := os.Stat(g.folder); os.IsNotExist(err) {
		// #nosec
		args := []string{
			"clone",
			"--recursive",
			"--depth", "1",
		}
		if g.Version != "" {
			args = append(args, "-b", g.Version)
		}
		args = append(args, g.URL, g.folder)
		var cmd = exec.Command("git", args...)
		cmd.Env = gitCmdEnv

		if bts, err := cmd.CombinedOutput(); err != nil {
			log.Println("git clone failed for", g.URL, string(bts))
			return err
		}
	}
	return nil
}

func (g gitProject) Update() error {
	log.Println("updating:", g.URL)
	oldRev, err := commit(g.folder)
	if err != nil {
		return err
	}
	// #nosec
	args := []string{
		"pull",
		"--recurse-submodules",
		"origin",
	}
	if g.Version != "" {
		args = append(args, g.Version)
	}
	cmd := exec.Command("git", args...)
	cmd.Env = gitCmdEnv

	cmd.Dir = g.folder
	if bts, err := cmd.CombinedOutput(); err != nil {
		log.Println("git update failed for", g.folder, string(bts))
		return err
	}
	rev, err := commit(g.folder)
	if err != nil {
		return err
	}
	if rev != oldRev {
		log.Println("updated:", g.URL, oldRev, "->", rev)
	}
	return nil
}

func commit(folder string) (string, error) {
	// #nosec
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	cmd.Dir = folder
	rev, err := cmd.Output()
	return strings.ReplaceAll(string(rev), "\n", ""), err
}

func branch(folder string) (string, error) {
	// #nosec
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = folder
	branch, err := cmd.Output()
	return strings.ReplaceAll(string(branch), "\n", ""), err
}

func (g gitProject) Path() string {
	return filepath.Join(g.folder, g.inner)
}
