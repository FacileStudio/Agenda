// Package kit is the sign-in glue an app with its own users table used to
// write by hand around porte: the request identity, the porte.UserStore over
// that table, the middleware that hydrates the identity, the GDPR erase hook,
// the picture URL backfill and the route mount.
//
// It is opt-in and adds nothing to porte's contract. Mount delegates to
// session.Manager.Mount and oidc.Kit.Mount and mounts no route of its own;
// the kit changes no status code, cookie, CSRF rule or /auth/config body, and
// it never touches pg.Schema.
//
// What stays in the app: the users table and its migrations, the /auth/me
// response shape, any column the kit does not name (a display colour, a
// billing rate), the rows an erased person owns, the directory state store,
// and every authorization rule beyond "is signed in" and "is_admin".
package kit
