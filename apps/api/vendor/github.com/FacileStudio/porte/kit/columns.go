package kit

// Columns names the app's users table and the columns the kit reads and
// writes. A zero field takes the default DefaultColumns gives it, except
// Admin: an empty Admin means the table has no admin flag, so the first
// account gets no role and every Identity reads IsAdmin false.
//
// CreatedAt and UpdatedAt name timestamp columns the kit fills itself: now()
// into both when it creates an account, now() into UpdatedAt on every update
// it makes. Left empty, NewStore adopts a created_at or updated_at column that
// has no database default, which is what a GORM-created table looks like.
type Columns struct {
	Table     string
	ID        string
	Email     string
	Name      string
	Picture   string
	Locale    string
	Admin     string
	CreatedAt string
	UpdatedAt string
}

// DefaultColumns is the schema most suite apps share: users(id, email, name,
// avatar_url, locale), with no admin flag.
func DefaultColumns() Columns {
	return Columns{Table: "users", ID: "id", Email: "email", Name: "name", Picture: "avatar_url", Locale: "locale"}
}

func (c Columns) resolved() Columns {
	defaults := DefaultColumns()
	c.Table = fallback(c.Table, defaults.Table)
	c.ID = fallback(c.ID, defaults.ID)
	c.Email = fallback(c.Email, defaults.Email)
	c.Name = fallback(c.Name, defaults.Name)
	c.Picture = fallback(c.Picture, defaults.Picture)
	c.Locale = fallback(c.Locale, defaults.Locale)
	return c
}

func fallback(value, otherwise string) string {
	if value == "" {
		return otherwise
	}
	return value
}

func (c Columns) inserted() []string {
	return append([]string{c.Email, c.Name, c.Picture, c.Locale}, present(c.Admin, c.CreatedAt, c.UpdatedAt)...)
}

func (c Columns) touch() string {
	if c.UpdatedAt == "" {
		return ""
	}
	return ", " + quote(c.UpdatedAt) + " = now()"
}

func present(names ...string) []string {
	var out []string
	for _, name := range names {
		if name != "" {
			out = append(out, name)
		}
	}
	return out
}
