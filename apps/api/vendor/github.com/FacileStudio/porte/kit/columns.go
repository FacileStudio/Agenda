package kit

// Columns names the app's users table and the columns the kit reads and
// writes. A zero field takes the default DefaultColumns gives it, except
// Admin: an empty Admin means the table has no admin flag, so the first
// account gets no role and every Identity reads IsAdmin false.
type Columns struct {
	Table   string
	ID      string
	Email   string
	Name    string
	Picture string
	Locale  string
	Admin   string
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
