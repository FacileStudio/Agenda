// Package handoff renders the two pages porte puts in front of a human itself.
//
// Almost nothing in a login is porte's to draw. A successful callback redirects
// to OIDC_SUCCESS_URL and a failed one to the app's own login page, which is
// what Config.LoginFailure exists for: an app owns its own chrome, and a
// library that renders over it is a library that ships a second design system
// nobody asked for.
//
// Two moments cannot be handed back that way. A CLI login with no loopback
// listener has to show the one-time code somewhere, and a CLI's own loopback
// listener has to say something once the browser lands on 127.0.0.1. Neither
// has an app page to redirect to. Both were unstyled defaults until this
// package, which is the divergence it closes: the failure path had been given
// a real page and the success path had not.
//
// The markup lives in one package because it is served from two processes.
// porte/oidc serves it from the app's API and porte/loopback serves it from
// inside every CLI binary, and two copies of a page become two pages the first
// time only one of them is edited.
//
// It depends on the standard library and nothing else, because porte/loopback
// is linked into CLI binaries. A CLI that opens a browser and waits on a socket
// must not compile go-oidc, oauth2, chi or a database driver to do it.
package handoff

import (
	"html/template"
	"net/http"
)

// Page is everything the template renders. It is one struct for both pages
// because they are one page: a refusal that looked different from a success
// would be a second thing to keep in step, and the only real difference is a
// colour and a heading.
type Page struct {
	// AppName is the tool the page belongs to, "Courrier" or "Journal". A
	// page that names nobody is one a user cannot tell from any other
	// process that asked them to sign in, so it is never empty by the time
	// it reaches the template.
	AppName string

	// LogoURL is the image beside the name. Empty draws no image at all,
	// and an image that fails to load removes itself, so neither a missing
	// file nor an app that serves no logo leaves a broken icon behind.
	LogoURL string

	// Heading and Body are the message. Body carries the sentence a user
	// acts on; Hint is the smaller line under the card, for what happens
	// next rather than what just happened.
	Heading string
	Body    string
	Hint    string

	// Code is the one-time login code, shown with a copy button. Empty
	// omits the block entirely, which is what every page but the code page
	// wants.
	Code string

	// Warn colours the body as a refusal. It is presentation only: the
	// status code is the caller's, and nothing here decides it.
	Warn bool
}

// Write renders page to w under status.
//
// It sets the content type and stops there. Cache-Control belongs to the
// caller because only the caller knows whether the page carries a credential:
// the code page does and answers no-store, as OAuth 2.1 §7.1 requires of any
// response carrying one, and a loopback page does not.
//
// A render that fails has already written part of a page, so there is nothing
// left to say to the browser and no error worth returning to a handler that
// could not act on it either.
func Write(w http.ResponseWriter, status int, page Page) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = tmpl.Execute(w, page)
}

// tmpl is the markup. It is html/template rather than text/template because
// AppName and Code reach it from configuration and from a token generator, and
// contextual escaping is what keeps either from ending the script block that
// follows them.
//
// The copy button degrades on purpose: navigator.clipboard is unavailable over
// plain http on every browser but localhost, and the code page is reached over
// https while the loopback pages are localhost, so the textarea fallback is
// the path that runs when neither holds.
var tmpl = template.Must(template.New("handoff").Parse(`<!doctype html>
<html lang="en">
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>{{.Heading}} · {{.AppName}}</title>
<style>
  :root{color-scheme:light dark;--bg:#fafafa;--fg:#18181b;--muted:#71717a;--card:#fff;--line:#e4e4e7;--btn:#18181b;--btn-fg:#fafafa;--warn:#b45309}
  @media (prefers-color-scheme:dark){:root{--bg:#09090b;--fg:#fafafa;--muted:#a1a1aa;--card:#18181b;--line:#27272a;--btn:#fafafa;--btn-fg:#18181b;--warn:#fbbf24}}
  *{box-sizing:border-box}
  body{margin:0;min-height:100dvh;display:flex;align-items:center;justify-content:center;padding:1.5rem;
       background:var(--bg);color:var(--fg);
       font:16px/1.6 ui-sans-serif,system-ui,-apple-system,"Segoe UI",Roboto,sans-serif}
  main{width:100%;max-width:26rem}
  .brand{display:flex;align-items:center;justify-content:center;gap:.625rem;margin-bottom:2rem}
  .brand img{display:block;width:28px;height:28px}
  .brand span{font-size:1.125rem;font-weight:650;letter-spacing:-.01em}
  .card{background:var(--card);border:1px solid var(--line);border-radius:.75rem;padding:1.75rem;text-align:center}
  h1{margin:0;font-size:1.25rem;font-weight:650;letter-spacing:-.01em}
  p{margin:.5rem 0 0;font-size:.875rem;color:var(--muted)}
  p.warn{color:var(--warn)}
  .code{display:flex;align-items:center;gap:.5rem;margin-top:1.25rem}
  .code output{flex:1;min-width:0;padding:.75rem;border:1px solid var(--line);border-radius:.5rem;
                background:var(--bg);font-family:ui-monospace,SFMono-Regular,Menlo,monospace;
                font-size:1.0625rem;letter-spacing:.15em;word-break:break-all}
  button{flex:none;height:2.75rem;padding:0 1rem;border:0;border-radius:.5rem;background:var(--btn);color:var(--btn-fg);
         font:inherit;font-size:.875rem;font-weight:500;cursor:pointer}
  button:hover{opacity:.9}
  .hint{margin-top:1.5rem;text-align:center;font-size:.8125rem;color:var(--muted)}
</style>
<main>
  <div class="brand">
    {{if .LogoURL}}<img src="{{.LogoURL}}" alt="" onerror="this.remove()">{{end}}
    <span>{{.AppName}}</span>
  </div>
  <div class="card">
    <h1>{{.Heading}}</h1>
    <p{{if .Warn}} class="warn"{{end}}>{{.Body}}</p>
    {{if .Code}}
    <div class="code">
      <output id="c">{{.Code}}</output>
      <button type="button" id="b">Copy</button>
    </div>
    {{end}}
  </div>
  <p class="hint">{{.Hint}}</p>
</main>
{{if .Code}}
<script>
  const b=document.getElementById("b"),c=document.getElementById("c");
  b.addEventListener("click",async()=>{
    const t=c.textContent.trim();
    try{await navigator.clipboard.writeText(t)}
    catch{const s=document.createElement("textarea");s.value=t;document.body.append(s);s.select();document.execCommand("copy");s.remove()}
    b.textContent="Copied";setTimeout(()=>b.textContent="Copy",1600);
  });
</script>
{{end}}
`))
