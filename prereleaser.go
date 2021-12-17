package sdlc

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/pkg/errors"
	"github.com/tsuyoshiwada/go-gitlog"
)

type GitApp struct {
	repository      *git.Repository
	repoPath        string
	head            *plumbing.Reference
	latestTag       string
	latestTagCommit *object.Commit
	trunkBranch     string
}

type GitAppAuthOptions struct {
	SSHPrivateKeyFile string
	SSHPassphrase     string
}

func NewGitApp() *GitApp {
	return &GitApp{
		trunkBranch: "master",
	}
}

func NewGitAppFromRepository(repository *git.Repository, repositoryLocalPath string) (*GitApp, error) {
	gitApp := NewGitApp()
	gitApp.repository = repository
	gitApp.repoPath = repositoryLocalPath

	head, err := repository.Head()
	if err != nil {
		return nil, errors.Wrap(err, "error getting head")
	}

	gitApp.head = head

	err = gitApp.getLatestTag()
	if err != nil {
		return nil, errors.Wrap(err, "error getting latest tag")
	}

	log.Println("latest tag: ", gitApp.latestTag, gitApp.latestTagCommit)

	// err = gitApp.updateLocalStatus()
	// if err != nil {
	// 	return nil, errors.Wrap(err, "error updating local status")
	// }

	return gitApp, nil
}

func NewGitAppFromDir(dir string) (*GitApp, error) {
	repository, err := git.PlainOpen(dir)
	if err != nil {
		return nil, errors.Wrapf(err, "error opening git repository on path %s", dir)
	}

	gitApp, err := NewGitAppFromRepository(repository, dir)
	if err != nil {
		return nil, errors.Wrap(err, "error new git app from repository")
	}

	gitApp.repoPath = dir

	return gitApp, nil
}

func NewGitAppFromRemote(repoURL string, auth *GitAppAuthOptions) (*GitApp, error) {
	tempDir := filepath.Join("/tmp", repoURL)
	err := os.MkdirAll(tempDir, os.ModePerm)
	if err != nil {
		return nil, errors.Wrapf(err, "error creating temp directory %s", tempDir)
	}

	publicKeys, err := ssh.NewPublicKeysFromFile("git", auth.SSHPrivateKeyFile, auth.SSHPassphrase)
	if err != nil {
		return nil, errors.Wrap(err, "error on new public keys from file")
	}

	repoURLParts := strings.Split(repoURL, "/")
	remoteURL := fmt.Sprintf("git@%s:/%s/%s", repoURLParts[0], repoURLParts[1], repoURLParts[2])

	repository, err := git.PlainClone(tempDir, false, &git.CloneOptions{
		URL:  remoteURL,
		Auth: publicKeys,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "error cloning git repository %s", repoURL)
	}

	return NewGitAppFromRepository(repository, tempDir)
}

func (g *GitApp) updateLocalStatus() error {
	worktree, err := g.repository.Worktree()
	if err != nil {
		return errors.Wrap(err, "error getting worktree")
	}
	err = worktree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.Master,
	})
	if err != nil {
		return errors.Wrap(err, "error when checking out to trunk branch")
	}
	// make sure trunk branch
	err = worktree.Pull(&git.PullOptions{
		ReferenceName: plumbing.Master,
		SingleBranch:  true,
	})
	if err != nil {
		return errors.Wrap(err, "error when pulling trunk branch")
	}

	return nil
}

func (g *GitApp) getLatestTag() error {
	tagRefs, err := g.repository.Tags()
	if err != nil {
		return errors.Wrap(err, "error on get tags")
	}

	var latestTagCommit *object.Commit
	var latestTagName string
	err = tagRefs.ForEach(func(tagRef *plumbing.Reference) error {
		revision := plumbing.Revision(tagRef.Name().String())
		tagCommitHash, err := g.repository.ResolveRevision(revision)
		if err != nil {
			return errors.Wrapf(err, "error on resolving revision %s", revision)
		}

		commit, err := g.repository.CommitObject(*tagCommitHash)
		if err != nil {
			return errors.Wrapf(err, "error on getting commit object for ref %s", revision)
		}

		if latestTagCommit == nil {
			latestTagCommit = commit
			latestTagName = tagRef.Name().String()
		}

		if commit.Committer.When.After(latestTagCommit.Committer.When) {
			latestTagCommit = commit
			latestTagName = tagRef.Name().String()
		}

		return nil
	})
	if err != nil {
		return err
	}

	g.latestTag = latestTagName
	g.latestTagCommit = latestTagCommit

	return nil
}

func (g *GitApp) GetLatestChangeLogs() (*ChangelogList, error) {
	return g.GetChangeLogs(g.head.Hash(), g.latestTagCommit.Hash)
}

func (g *GitApp) GetChangeLogs(from, to plumbing.Hash) (*ChangelogList, error) {
	gitLog := gitlog.New(&gitlog.Config{
		Path: g.repoPath,
	})
	commits, err := gitLog.Log(&gitlog.RevRange{
		Old: g.latestTag,
		New: g.trunkBranch,
	}, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error getting git log from latest tag")
	}

	changelogList := NewChangelogList(&ChangelogListOptions{
		JiraURL: "https://tokopedia.atlassian.net",
	})

	for _, commit := range commits {
		changelog := ChangelogFromCommitMessage(commit.Subject)
		if changelog == nil {
			continue
		}
		changelog.Hash = commit.Hash

		changelogList.Add(changelog)
	}

	return &changelogList, nil
}

func (g *GitApp) GenerateSchemaScript() (string, error) {
	headCommit, err := g.repository.CommitObject(g.head.Hash())
	if err != nil {
		return "", errors.Wrap(err, "error getting head commit")
	}

	headTree, err := headCommit.Tree()
	if err != nil {
		return "", errors.Wrap(err, "error getting head tree")
	}

	tree, err := g.latestTagCommit.Tree()
	if err != nil {
		return "", errors.Wrap(err, "error getting latest tagged commit tree")
	}

	changes, err := headTree.Diff(tree)
	if err != nil {
		return "", errors.Wrap(err, "error getting changes from latest tag")
	}

	patch, err := changes.Patch()
	if err != nil {
		return "", errors.Wrap(err, "error get diff patch")
	}

	var schemaChanges []string
	for _, filePatch := range patch.FilePatches() {
		from, _ := filePatch.Files()
		if from == nil {
			continue
		}
		if strings.HasSuffix(from.Path(), ".sql") {
			schemaChanges = append(schemaChanges, from.Path())
		}
	}

	var schemaChangesBuffer strings.Builder
	for _, schemaChange := range schemaChanges {
		schemaFile, err := os.Open(filepath.Join(g.repoPath, schemaChange))
		if err != nil {
			log.Fatalf("error opening schema change %s: %v", schemaChange, err)
		}
		defer schemaFile.Close()

		schemaChangesBuffer.WriteString("-- " + schemaChange)
		schemaChangesBuffer.WriteString("\n")
		io.Copy(&schemaChangesBuffer, schemaFile)
		schemaChangesBuffer.WriteString("\n\n")
	}

	return schemaChangesBuffer.String(), nil
}
