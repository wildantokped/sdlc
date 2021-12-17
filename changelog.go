package sdlc

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/tsuyoshiwada/go-gitlog"
)

const (
	CommitTypeFeat    = "feat"
	CommitTypeFeature = "feature"
	CommitTypeFix     = "fix"
	CommitTypeOther   = "other"
)

var (
	conventionalCommitRegexStr = `^(?P<type>feature|feat|fix|docs|style|refactor|test|chore|enhance|config)(?P<scope>(?:\([^()\r\n]*\)|\())?(?P<breaking>!)?(?P<separator>:)? *(?P<ticket>\[[A-Z][A-Z]+\-[0-9]+\])? *(?P<message>.+$)?`
)

type Changelog struct {
	Type       string
	Scope      string
	Message    string
	Breaking   bool
	JiraTicket string
	Hash       *gitlog.Hash
}

func ChangelogFromCommitMessage(msg string) *Changelog {
	msg = strings.TrimSpace(msg)
	conventionalCommitRegex := regexp.MustCompile(conventionalCommitRegexStr)
	match := conventionalCommitRegex.FindStringSubmatch(msg)
	if len(match) == 0 {
		return nil
	}
	result := make(map[string]string)
	for i, name := range conventionalCommitRegex.SubexpNames() {
		if i != 0 && name != "" {
			result[name] = match[i]
		}
	}
	message := result["message"]
	if message == "" {
		// if message is nil
		return nil
	}
	// remove () from scope
	scope := result["scope"]
	if len(scope) > 2 {
		scope = scope[1 : len(scope)-1]
	}
	// remove [] from ticket
	ticket := result["ticket"]
	if len(ticket) > 2 {
		ticket = ticket[1 : len(ticket)-1]
	}
	return &Changelog{
		Type:       strings.ToLower(result["type"]),
		Scope:      strings.ToLower(scope),
		Message:    result["message"],
		Breaking:   result["breaking"] == "!",
		JiraTicket: ticket,
	}
}

type ChangelogList struct {
	changelogs []*Changelog
	opt        *ChangelogListOptions
}

type ChangelogListOptions struct {
	Scope   string
	JiraURL string
}

func NewChangelogList(opt *ChangelogListOptions) ChangelogList {
	return ChangelogList{
		opt: opt,
	}
}

func (c *ChangelogList) Add(changelog *Changelog) {
	c.changelogs = append(c.changelogs, changelog)
}

func (c *ChangelogList) FormatAsMarkdown() string {
	var markdown strings.Builder

	groupByScopeAndCommitType := map[string]map[string][]*Changelog{}
	for _, changelog := range c.changelogs {
		if c.opt.Scope != "" && changelog.Scope != c.opt.Scope {
			// is scope is set, only get changelogs from that scope
			continue
		}
		if groupByScopeAndCommitType[changelog.Scope] == nil {
			groupByScopeAndCommitType[changelog.Scope] = map[string][]*Changelog{}
		}
		commitType := changelog.Type
		switch commitType {
		case CommitTypeFeature:
		case CommitTypeFeat:
			commitType = CommitTypeFeature
		case CommitTypeFix:
		default:
			commitType = CommitTypeOther
		}
		groupByScopeAndCommitType[changelog.Scope][commitType] = append(groupByScopeAndCommitType[changelog.Scope][commitType], changelog)
	}

	for scope, groupByCommitType := range groupByScopeAndCommitType {
		markdown.WriteString("### " + scope)
		// generate commit type table
		markdown.WriteString("\n\n| **New Features** | **Bug Fixes** | **Others** |\n| --- | --- | --- |\n")
		var featDone, fixDone, othersDone bool
		var (
			features     = groupByCommitType[CommitTypeFeature]
			fixes        = groupByCommitType[CommitTypeFix]
			otherChanges = groupByCommitType[CommitTypeOther]
		)
		for row := 0; ; row++ {
			if row == len(features) {
				featDone = true
			}
			if row == len(fixes) {
				fixDone = true
			}
			if row == len(otherChanges) {
				othersDone = true
			}
			if featDone && fixDone && othersDone {
				break
			}
			if !featDone {
				c.writeChangelogRow(&markdown, features[row], featDone, false)
			}
			if !fixDone {
				c.writeChangelogRow(&markdown, fixes[row], fixDone, false)
			}
			if !othersDone {
				c.writeChangelogRow(&markdown, otherChanges[row], othersDone, true)
			}
			markdown.WriteString("|\n")
		}
		markdown.WriteString("\n\n")
	}
	return markdown.String()
}

func (c *ChangelogList) String() string {
	return c.FormatAsMarkdown()
}

func (c *ChangelogList) writeChangelogRow(markdown *strings.Builder, changelog *Changelog, done bool, prefixType bool) {
	markdown.WriteString("|")
	if done {
		markdown.WriteString(" ")
	} else {
		commitMessage := fmt.Sprintf(" %s ", changelog.Message)
		if prefixType {
			commitMessage = fmt.Sprintf("%s: %s ", changelog.Type, changelog.Message)
		}
		if changelog.JiraTicket != "" {
			jiraTicket := changelog.JiraTicket
			commitMessage += fmt.Sprintf("([%s](%s/browse/%s)) ", jiraTicket, c.opt.JiraURL, jiraTicket)
		}
		markdown.WriteString(commitMessage)
	}
}
