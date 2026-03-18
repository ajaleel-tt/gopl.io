// Exercise 4.14: Web server that queries GitHub once and serves issue data.
package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"gopl.io/ch4/github"
)

var result *github.IssuesSearchResult

var indexTempl = template.Must(template.New("index").Parse(`
<html>
<head><title>GitHub Issues</title></head>
<body>
<h1>{{.TotalCount}} Issues</h1>
<div style="display: flex">
<div style="flex: 3">
<table>
<tr style='text-align: left'>
  <th>#</th>
  <th>State</th>
  <th>User</th>
  <th>Milestone</th>
  <th>Title</th>
</tr>
{{range .Items}}
<tr>
  <td><a href='{{.HTMLURL}}'>{{.Number}}</a></td>
  <td>{{.State}}</td>
  <td><a href='/user?login={{.User.Login}}'>{{.User.Login}}</a></td>
  <td>{{if .Milestone}}<a href='/milestone?name={{.Milestone.Title}}'>{{.Milestone.Title}}</a>{{end}}</td>
  <td><a href='{{.HTMLURL}}'>{{.Title}}</a></td>
</tr>
{{end}}
</table>
</div>
<div style="flex: 1; padding-left: 2em">
<h2>Milestones</h2>
<ul>
{{range .Milestones}}<li><a href='/milestone?name={{.}}'>{{.}}</a></li>
{{end}}
</ul>
<h2>Users</h2>
<ul>
{{range .Users}}<li><a href='/user?login={{.}}'>{{.}}</a></li>
{{end}}
</ul>
</div>
</div>
</body>
</html>
`))

var milestoneTempl = template.Must(template.New("milestone").Parse(`
<html>
<head><title>Milestone: {{.Name}}</title></head>
<body>
<h1>Milestone: {{.Name}}</h1>
<p><a href="/">Back to all issues</a></p>
<table>
<tr style='text-align: left'>
  <th>#</th>
  <th>State</th>
  <th>User</th>
  <th>Title</th>
</tr>
{{range .Issues}}
<tr>
  <td><a href='{{.HTMLURL}}'>{{.Number}}</a></td>
  <td>{{.State}}</td>
  <td><a href='/user?login={{.User.Login}}'>{{.User.Login}}</a></td>
  <td><a href='{{.HTMLURL}}'>{{.Title}}</a></td>
</tr>
{{end}}
</table>
</body>
</html>
`))

var userTempl = template.Must(template.New("user").Parse(`
<html>
<head><title>User: {{.Login}}</title></head>
<body>
<h1>User: {{.Login}}</h1>
<p><a href="/">Back to all issues</a></p>
<table>
<tr style='text-align: left'>
  <th>#</th>
  <th>State</th>
  <th>Milestone</th>
  <th>Title</th>
</tr>
{{range .Issues}}
<tr>
  <td><a href='{{.HTMLURL}}'>{{.Number}}</a></td>
  <td>{{.State}}</td>
  <td>{{if .Milestone}}{{.Milestone.Title}}{{end}}</td>
  <td><a href='{{.HTMLURL}}'>{{.Title}}</a></td>
</tr>
{{end}}
</table>
</body>
</html>
`))

type IndexData struct {
	TotalCount int
	Items      []*github.Issue
	Milestones []string
	Users      []string
}

type MilestoneData struct {
	Name   string
	Issues []*github.Issue
}

type UserData struct {
	Login  string
	Issues []*github.Issue
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: issuesweb <search terms>")
		os.Exit(1)
	}

	var err error
	result, err = github.SearchIssues(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Loaded %d issues. Starting server on :8080\n", result.TotalCount)

	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/milestone", handleMilestone)
	http.HandleFunc("/user", handleUser)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func uniqueMilestones() []string {
	seen := make(map[string]bool)
	var names []string
	for _, issue := range result.Items {
		if issue.Milestone != nil && !seen[issue.Milestone.Title] {
			seen[issue.Milestone.Title] = true
			names = append(names, issue.Milestone.Title)
		}
	}
	return names
}

func uniqueUsers() []string {
	seen := make(map[string]bool)
	var logins []string
	for _, issue := range result.Items {
		if issue.User != nil && !seen[issue.User.Login] {
			seen[issue.User.Login] = true
			logins = append(logins, issue.User.Login)
		}
	}
	return logins
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	data := IndexData{
		TotalCount: result.TotalCount,
		Items:      result.Items,
		Milestones: uniqueMilestones(),
		Users:      uniqueUsers(),
	}
	if err := indexTempl.Execute(w, data); err != nil {
		log.Print(err)
	}
}

func handleMilestone(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "missing milestone name", http.StatusBadRequest)
		return
	}
	var issues []*github.Issue
	for _, issue := range result.Items {
		if issue.Milestone != nil && issue.Milestone.Title == name {
			issues = append(issues, issue)
		}
	}
	data := MilestoneData{Name: name, Issues: issues}
	if err := milestoneTempl.Execute(w, data); err != nil {
		log.Print(err)
	}
}

func handleUser(w http.ResponseWriter, r *http.Request) {
	login := r.URL.Query().Get("login")
	if login == "" {
		http.Error(w, "missing user login", http.StatusBadRequest)
		return
	}
	var issues []*github.Issue
	for _, issue := range result.Items {
		if issue.User != nil && issue.User.Login == login {
			issues = append(issues, issue)
		}
	}
	data := UserData{Login: login, Issues: issues}
	if err := userTempl.Execute(w, data); err != nil {
		log.Print(err)
	}
}
