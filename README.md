# SDLC Tools
The goal is to build a suite of tools to automate mundane SDLC tasks, like writing a release document, before release.

Roadmap:

- [x] Prereleaser, generate release documents
- [ ] Automatically request Jira tickets on database changes
- [ ] etc.

## Prereleaser

### Installation

```
go get github.com/wildantokped/sdlc/cmd/sdlc
```

or Go 1.17+

```
go install github.com/wildantokped/sdlc/cmd/sdlc
```

### Usage

1. Open the root of your working repository through your terminal.

2. Make sure to checkout to your repositories trunk branch, for example if your trunk branch is `master`.

```
git checkout master
```

3. Run the following command:

```
sdlc .
```

4. It will generate a markdown file `CHANGELOG-<timestamp>.md`, copy and paste the markdown into confluence. For Mac OS, Paste the markdown with the `Paste and Match Style` command. For Ubuntu, you must `Paste as plain text`.

5. After release, please make sure a git tag is created after the deployment, since this tool will compare with the latest tag in the repository.
