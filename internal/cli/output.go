package cli

import (
	"encoding/json"
	"fmt"
	"io"
)

func WriteJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func WriteLine(w io.Writer, format string, args ...any) {
	fmt.Fprintf(w, format+"\n", args...)
}
