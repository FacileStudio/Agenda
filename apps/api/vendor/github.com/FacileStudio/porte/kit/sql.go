package kit

import (
	"strconv"
	"strings"
)

func quote(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func text(column string) string {
	return "coalesce(" + quote(column) + ", '')"
}

func (c Columns) identitySQL() string {
	admin := "false"
	if c.Admin != "" {
		admin = quote(c.Admin)
	}
	return "SELECT " + quote(c.ID) + ", " + text(c.Email) + ", " + text(c.Name) + ", " + text(c.Picture) + ", " +
		admin + " FROM " + quote(c.Table) + " WHERE " + quote(c.ID) + " = $1"
}

func (c Columns) linkedSQL() string {
	return "SELECT u." + quote(c.ID) + " FROM porte_identities i JOIN " + quote(c.Table) + " u ON u." + quote(c.ID) +
		" = i.user_id WHERE i.provider = $1 AND i.subject = $2"
}

func (c Columns) byEmailSQL() string {
	return "SELECT " + quote(c.ID) + " FROM " + quote(c.Table) + " WHERE lower(" + quote(c.Email) + ") = $1 ORDER BY " +
		quote(c.ID) + " LIMIT 1"
}

func (c Columns) insertSQL() string {
	columns := []string{quote(c.Email), quote(c.Name), quote(c.Picture), quote(c.Locale)}
	if c.Admin != "" {
		columns = append(columns, quote(c.Admin))
	}
	placeholders := make([]string, len(columns))
	for i := range columns {
		placeholders[i] = "$" + strconv.Itoa(i+1)
	}
	return "INSERT INTO " + quote(c.Table) + " (" + strings.Join(columns, ", ") + ") VALUES (" +
		strings.Join(placeholders, ", ") + ") RETURNING " + quote(c.ID)
}

func (c Columns) updateSQL(userID int64, p profile) (string, []any) {
	sets := []string{quote(c.Picture) + " = $2"}
	args := []any{userID, p.picture}
	for _, field := range [][2]string{{c.Email, p.verifiedEmail()}, {c.Name, p.name}, {c.Locale, p.locale}} {
		if field[1] == "" {
			continue
		}
		args = append(args, field[1])
		sets = append(sets, quote(field[0])+" = $"+strconv.Itoa(len(args)))
	}
	return "UPDATE " + quote(c.Table) + " SET " + strings.Join(sets, ", ") + " WHERE " + quote(c.ID) + " = $1", args
}

func (c Columns) backfillSQL() string {
	return "UPDATE " + quote(c.Table) + " u SET " + quote(c.Picture) + " = $1 || i.subject FROM porte_identities i" +
		" WHERE i.user_id = u." + quote(c.ID) + " AND rtrim(i.provider, '/') = $2 AND coalesce(u." + quote(c.Picture) + ", '') = ''"
}
