package mcp

// Offset reads the position of an offset cursor made by Cursor: 0 when the
// cursor is empty, ErrCursor when it is malformed or negative.
func Offset(cursor string) (int, error) {
	offset := 0
	if err := DecodeCursor(cursor, &offset); err != nil || offset < 0 {
		return 0, ErrCursor
	}
	return offset, nil
}

// Cursor is the cursor of the page starting at offset in a list of total
// items, empty when offset is past the end.
func Cursor(offset, total int) string {
	if offset >= total {
		return ""
	}
	return EncodeCursor(offset)
}

// Page cuts the page of at most size items, at least one, at the cursor's offset and
// returns the cursor of the next page, empty on the last one.
func Page[T any](items []T, cursor string, size int) ([]T, string, error) {
	offset, err := Offset(cursor)
	if err != nil {
		return nil, "", err
	}
	offset = min(offset, len(items))
	end := min(offset+max(size, 1), len(items))
	return items[offset:end], Cursor(end, len(items)), nil
}
