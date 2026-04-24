package main

import (
	"embed"
	"log/slog"
	"os"
	"strings"
)

//go:embed snippets/*.txt
var snippetFS embed.FS

//go:embed static
var staticFS embed.FS

func mustSnippet(name string) string {
	b, err := snippetFS.ReadFile("snippets/" + name + ".txt")
	if err != nil {
		slog.Error("missing snippet file", "name", name, "err", err)
		os.Exit(1)
	}
	return strings.TrimRight(string(b), "\n")
}

// Source snippets, loaded once at package init from cmd/design.within.website/snippets.
var (
	typeSrc         = mustSnippet("type")
	typeBodySrc     = mustSnippet("type-body")
	buttonsSrc      = mustSnippet("buttons")
	buttonLinkSrc   = mustSnippet("button-link")
	cardSrc         = mustSnippet("card")
	tagListSrc      = mustSnippet("tag-list")
	badgesSrc       = mustSnippet("badges")
	textScaleSrc    = mustSnippet("text-scale")
	surfaceSrc      = mustSnippet("surface")
	dividersSrc     = mustSnippet("dividers")
	quoteSrc        = mustSnippet("quote")
	pullSrc         = mustSnippet("pull")
	preSrc          = mustSnippet("pre")
	detailsSrc      = mustSnippet("details")
	markKbdSrc      = mustSnippet("mark-kbd")
	figureSrc       = mustSnippet("figure")
	admonInfoSrc    = mustSnippet("admon-info")
	admonWarnSrc    = mustSnippet("admon-warn")
	admonTipSrc     = mustSnippet("admon-tip")
	admonNoteSrc    = mustSnippet("admon-note")
	admonDangerSrc  = mustSnippet("admon-danger")
	admonSuccessSrc = mustSnippet("admon-success")
	chatSrc         = mustSnippet("chat")
	sponsorSrc      = mustSnippet("sponsor")
	linksSrc        = mustSnippet("links")
	iconSrc         = mustSnippet("icon")
	spinnerSrc      = mustSnippet("spinner")
	breadcrumbSrc   = mustSnippet("breadcrumb")
	paginationSrc   = mustSnippet("pagination")
	tocSrc          = mustSnippet("toc")
	nextprevSrc     = mustSnippet("nextprev")
	backtopSrc      = mustSnippet("backtop")
	formSrc         = mustSnippet("form")
	toggleSrc       = mustSnippet("toggle")
	selectSrc       = mustSnippet("select")
	helperSrc       = mustSnippet("helper")
	pageHeaderSrc   = mustSnippet("page-header")
	postCardSrc     = mustSnippet("post-card")
	toastSrc        = mustSnippet("toast")
	modalSrc        = mustSnippet("modal")
	progressSrc     = mustSnippet("progress")
	tooltipSrc      = mustSnippet("tooltip")
	statSrc         = mustSnippet("stat")
	dlSrc           = mustSnippet("dl")
	timelineSrc     = mustSnippet("timeline")
	tableSrc        = mustSnippet("table")
)
