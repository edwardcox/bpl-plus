package interpreter

// SetSource updates the interpreter's current "active" source context.
// This is used by the REPL so runtime errors show correct filename + caret lines,
// and imports resolve relative to the chunk filename (CWD-anchored).
func (i *Interpreter) SetSource(filename string, source string) {
	i.filename = filename
	i.lines = splitLinesPreserve(source)
}
