package spa

import (
	"crypto/sha256"
	"encoding/base64"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// InlineScriptHashes returns a CSP source, 'sha256-...', for every inline
// script in the built client's HTML files, sorted and without duplicates.
//
// It is how a SvelteKit client served by this binary passes an enforced
// script-src 'self'. adapter-static writes two inline scripts into every
// page: the bootstrap that imports the entry chunks, whose content changes on
// every build, and whatever app.html carries, such as a theme script. Hashing
// the files the binary is about to serve covers both, needs no SvelteKit
// configuration, and keeps the page cacheable, which a per-request nonce
// would not. Call it once at startup and pass the result as the script-src
// entry of SecurityConfig.CSPSources. Every hash lands in one header, so a
// build with more than about 70 distinct inline scripts outgrows what proxies
// pass; middleware.SecurityHeaders warns past middleware.MaxCSPHeaderBytes.
func InlineScriptHashes(dir string) ([]string, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()

	var hashes []string
	err = fs.WalkDir(root.FS(), ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(path), ".html") {
			return nil
		}
		page, err := fs.ReadFile(root.FS(), path)
		if err != nil {
			return err
		}
		hashes = append(hashes, hashInlineScripts(string(page))...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.Sort(hashes)
	return slices.Compact(hashes), nil
}

// hashInlineScripts hashes each script element without a src attribute, after
// the newline normalisation the HTML parser applies before the browser hashes.
func hashInlineScripts(page string) []string {
	page = strings.ReplaceAll(strings.ReplaceAll(page, "\r\n", "\n"), "\r", "\n")
	var hashes []string
	for _, script := range scanScripts(page) {
		if !script.inline() {
			continue
		}
		sum := sha256.Sum256([]byte(script.text))
		hashes = append(hashes, "'sha256-"+base64.StdEncoding.EncodeToString(sum[:])+"'")
	}
	return hashes
}

// asciiLower folds A-Z only, as the HTML tokenizer does, and keeps every byte
// offset: strings.ToLower resizes some runes (U+023A grows, U+212A and U+0130
// shrink), and indices found in its result would slice the wrong bytes of
// page, or past its end.
func asciiLower(value string) string {
	folded := []byte(value)
	for i, c := range folded {
		if 'A' <= c && c <= 'Z' {
			folded[i] = c + 'a' - 'A'
		}
	}
	return string(folded)
}
