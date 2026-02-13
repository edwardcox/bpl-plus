package interpreter

import "sort"

// GlobalsSnapshot returns a copy of global variables (sorted usage is caller-side).
func (i *Interpreter) GlobalsSnapshot() map[string]Value {
	out := make(map[string]Value, len(i.globals))
	for k, v := range i.globals {
		out[k] = v
	}
	return out
}

// FuncNames returns sorted names of user-defined functions.
func (i *Interpreter) FuncNames() []string {
	names := make([]string, 0, len(i.funcs))
	for name := range i.funcs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ModulesSnapshot returns module paths grouped by state.
// This is REPL-friendly and avoids exposing internal enums.
func (i *Interpreter) ModulesSnapshot() (loading []string, loaded []string) {
	for path, st := range i.modules {
		switch st {
		case modLoading:
			loading = append(loading, path)
		case modLoaded:
			loaded = append(loaded, path)
		default:
			// ignore modNone (shouldn't be present as a stored state typically)
		}
	}
	sort.Strings(loading)
	sort.Strings(loaded)
	return loading, loaded
}
