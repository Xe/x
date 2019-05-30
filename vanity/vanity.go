/*
Package vanity implements custom import paths (Go vanity URLs) as an HTTP
handler that can be installed at the vanity URL.
*/
package vanity

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
)

type config struct {
	importTag []string
	sourceTag *string
	redir     Redirector
}

// Configures the Handler. The only required option is WithImport.
type Option func(*config)

// Instructs the go tool where to fetch the repo at vcsRoot and the importPath
// that tree should be rooted at.
func WithImport(importPath, vcs, vcsRoot string) Option {
	importTag := "<meta name=\"go-import\" content=\"" + importPath + " " +
		vcs + " " + vcsRoot + "\">"
	return func(cfg *config) {
		cfg.importTag = append(cfg.importTag, importTag)
	}
}

// Instructs gddo (godoc.org) how to direct browsers to browsable source code
// for packages and their contents rooted at prefix.
//
// home specifies the home page of prefix, directory gives a format for how to
// browse a directory, and file gives a format for how to view a file and go to
// specific lines within it.
//
// More information can be found at https://github.com/golang/gddo/wiki/Source-Code-Links.
//
func WithSource(prefix, home, directory, file string) Option {
	sourceTag := "<meta name=\"go-source\" content=\"" + prefix + " " +
		home + " " + directory + " " + file + "\">"
	return func(cfg *config) {
		if cfg.sourceTag != nil {
			panic(fmt.Sprintf("vanity: existing source tag: %s", *cfg.sourceTag))
		}
		cfg.sourceTag = &sourceTag
	}
}

// When a browser navigates to the vanity URL of pkg, this function rewrites
// pkg to a browsable URL.
type Redirector func(pkg string) (url string)

// WithRedirector loads a redirector instance into the config.
func WithRedirector(redir Redirector) Option {
	return func(cfg *config) {
		if cfg.redir != nil {
			panic("vanity: existing Redirector")
		}
		cfg.redir = redir
	}
}

// MakeHandler creates a handler with custom options.
func MakeHandler(opts ...Option) http.Handler {
	return handlerFrom(compile(opts))
}

func compile(opts []Option) (*template.Template, Redirector) {
	// Process options.
	var cfg config
	for _, opt := range opts {
		opt(&cfg)
	}

	// A WithImport is required.
	if cfg.importTag == nil {
		panic("vanity: WithImport is required")
	}

	tags := make([]string, len(cfg.importTag))
	copy(tags, cfg.importTag)
	if cfg.sourceTag != nil {
		tags = append(tags, *cfg.sourceTag)
	}
	tagBlk := strings.Join(tags, "\n")

	h := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<meta http-equiv="Content-Type" content="text/html; charset=utf-8"/>
%s
<meta http-equiv="refresh" content="0; url={{ . }}">
</head>
<body>
Please see <a href="{{ . }}">here</a> for documentation on this package.
</body>
</html>
`, tagBlk)

	// Use default GDDO Redirector.
	if cfg.redir == nil {
		cfg.redir = func(pkg string) string {
			return "https://godoc.org/" + pkg
		}
	}

	return template.Must(template.New("").Parse(h)), cfg.redir
}

func handlerFrom(tpl *template.Template, redir Redirector) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only method supported is GET.
		if r.Method != http.MethodGet {
			status := http.StatusMethodNotAllowed
			http.Error(w, http.StatusText(status), status)
			return
		}

		pkg := r.Host + r.URL.Path
		redirURL := redir(pkg)

		// Issue an HTTP redirect if this is definitely a browser.
		if r.FormValue("go-get") != "1" {
			http.Redirect(w, r, redirURL, http.StatusTemporaryRedirect)
			return
		}

		w.Header().Set("Cache-Control", "public, max-age=300")
		if err := tpl.ExecuteTemplate(w, "", redirURL); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

// Handler returns an http.Handler that serves the vanity URL information for a single
// repository. Each Option gives additional information to agents about the
// repository or provides help to browsers that may have navigated to the vanity
// URL. The WithImport Option is mandatory since the go tool requires it to
// fetch the repository.
func Handler(opts ...Option) http.Handler {
	return handlerFrom(compile(opts))
}

// Helpers for common VCSs.

// Redirects gddo to browsable source files for GitHub hosted repositories.
func WithGitHubStyleSource(importPath, repoPath, ref string) Option {
	directory := repoPath + "/tree/" + ref + "{/dir}"
	file := repoPath + "/blob/" + ref + "{/dir}/{file}#L{line}"

	return WithSource(importPath, repoPath, directory, file)
}

// Redirects gddo to browsable source files for Gogs hosted repositories.
func WithGogsStyleSource(importPath, repoPath, ref string) Option {
	directory := repoPath + "/src/" + ref + "{/dir}"
	file := repoPath + "/src/" + ref + "{/dir}/{file}#L{line}"

	return WithSource(importPath, repoPath, directory, file)
}

// Creates a Handler that serves a GitHub repository at a specific importPath.
func GitHubHandler(importPath, user, repo, gitScheme string) http.Handler {
	ghImportPath := "github.com/" + user + "/" + repo
	return Handler(
		WithImport(importPath, "git", gitScheme+"://"+ghImportPath),
		WithGitHubStyleSource(importPath, "https://"+ghImportPath, "master"),
	)
}

// Creates a Handler that serves a repository hosted with Gogs at host at a
// specific importPath.
func GogsHandler(importPath, host, user, repo, gitScheme string) http.Handler {
	gogsImportPath := host + "/" + user + "/" + repo
	return Handler(
		WithImport(importPath, "git", gitScheme+"://"+gogsImportPath),
		WithGogsStyleSource(importPath, "https://"+gogsImportPath, "master"),
	)
}
