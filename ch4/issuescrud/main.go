// Exercise 4.11: Build a tool that lets users create, read, update, and close
// GitHub issues from the command line, invoking the user's preferred editor
// when substantial text input is required.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"gopl.io/ch4/github"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  issuescrud create owner repo\n")
	fmt.Fprintf(os.Stderr, "  issuescrud read   owner repo number\n")
	fmt.Fprintf(os.Stderr, "  issuescrud update owner repo number\n")
	fmt.Fprintf(os.Stderr, "  issuescrud close  owner repo number\n")
	os.Exit(1)
}

func getToken() string {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		fmt.Fprintf(os.Stderr, "GITHUB_TOKEN environment variable is not set.\n")
		fmt.Fprintf(os.Stderr, "Create a token at https://github.com/settings/tokens\n")
		os.Exit(1)
	}
	return token
}

func openEditor(title, body string) (string, string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	f, err := os.CreateTemp("", "issue-*.txt")
	if err != nil {
		return "", "", err
	}
	name := f.Name()
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			fmt.Fprintf(
				os.Stderr,
				"Warning: failed to remove temporary file %s: %v\n",
				name,
				err,
			)
		}
	}(name)

	template := fmt.Sprintf("Title: %s\n\n%s", title, body)
	if _, err := f.WriteString(template); err != nil {
		_ = f.Close()
		return "", "", err
	}
	_ = f.Close()

	cmd := exec.Command(editor, name)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", "", fmt.Errorf("editor exited with error: %v", err)
	}

	data, err := os.ReadFile(name)
	if err != nil {
		return "", "", err
	}

	lines := strings.SplitN(string(data), "\n\n", 2)
	newTitle := strings.TrimPrefix(lines[0], "Title: ")
	newTitle = strings.TrimSpace(newTitle)
	var newBody string
	if len(lines) > 1 {
		newBody = strings.TrimSpace(lines[1])
	}

	if newTitle == "" {
		return "", "", fmt.Errorf("title is empty; aborting")
	}

	return newTitle, newBody, nil
}

func main() {
	if len(os.Args) < 4 {
		usage()
	}

	cmd := os.Args[1]
	owner := os.Args[2]
	repo := os.Args[3]

	switch cmd {
	case "create":
		if len(os.Args) != 4 {
			usage()
		}
		token := getToken()
		title, body, err := openEditor("", "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		issue, err := github.CreateIssue(owner, repo, token, github.IssueRequest{
			Title: title,
			Body:  body,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating issue: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Created issue #%d: %s\n%s\n", issue.Number, issue.Title, issue.HTMLURL)

	case "read":
		if len(os.Args) != 5 {
			usage()
		}
		number, err := strconv.Atoi(os.Args[4])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid issue number: %s\n", os.Args[4])
			os.Exit(1)
		}
		issue, err := github.GetIssue(owner, repo, number)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading issue: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("#%d %s (%s)\n", issue.Number, issue.Title, issue.State)
		fmt.Printf("Author: %s\n", issue.User.Login)
		fmt.Printf("Created: %s\n", issue.CreatedAt.Format("2006-01-02 15:04:05"))
		if issue.Body != "" {
			fmt.Printf("\n%s\n", issue.Body)
		}

	case "update":
		if len(os.Args) != 5 {
			usage()
		}
		number, err := strconv.Atoi(os.Args[4])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid issue number: %s\n", os.Args[4])
			os.Exit(1)
		}
		token := getToken()
		issue, err := github.GetIssue(owner, repo, number)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching issue: %v\n", err)
			os.Exit(1)
		}
		title, body, err := openEditor(issue.Title, issue.Body)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		updated, err := github.UpdateIssue(owner, repo, token, number, github.IssueRequest{
			Title: title,
			Body:  body,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error updating issue: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Updated issue #%d: %s\n%s\n", updated.Number, updated.Title, updated.HTMLURL)

	case "close":
		if len(os.Args) != 5 {
			usage()
		}
		number, err := strconv.Atoi(os.Args[4])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid issue number: %s\n", os.Args[4])
			os.Exit(1)
		}
		token := getToken()
		issue, err := github.CloseIssue(owner, repo, token, number)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error closing issue: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Closed issue #%d: %s\n", issue.Number, issue.Title)

	default:
		usage()
	}
}
