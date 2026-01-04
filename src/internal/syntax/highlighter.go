package syntax

import (
	"regexp"
	"strings"

	"gti/src/internal/config"
)

// TokenType represents different types of syntax tokens
type TokenType int

const (
	TokenNormal TokenType = iota
	TokenKeyword
	TokenString
	TokenComment
	TokenNumber
	TokenOperator
	TokenBracket
	TokenTypeName
)

// Token represents a syntax token with its type and content
type Token struct {
	Type    TokenType
	Content string
}

// Highlighter provides syntax highlighting for different programming languages
type Highlighter struct {
	theme *config.Config
}

// NewHighlighter creates a new syntax highlighter
func NewHighlighter(cfg *config.Config) *Highlighter {
	return &Highlighter{theme: cfg}
}

// Highlight applies syntax highlighting to code based on language
func (h *Highlighter) Highlight(code, language string) string {
	switch strings.ToLower(language) {
	case "go":
		return h.highlightGo(code)
	case "python":
		return h.highlightPython(code)
	case "javascript", "typescript":
		return h.highlightJavaScript(code)
	case "java":
		return h.highlightJava(code)
	case "cpp":
		return h.highlightCpp(code)
	case "rust":
		return h.highlightRust(code)
	default:
		return code // No highlighting for unknown languages
	}
}

// highlightGo applies Go syntax highlighting
func (h *Highlighter) highlightGo(code string) string {
	// Go keywords
	keywords := []string{
		"break", "case", "chan", "const", "continue", "default", "defer", "else",
		"fallthrough", "for", "func", "go", "goto", "if", "import", "interface",
		"map", "package", "range", "return", "select", "struct", "switch", "type",
		"var", "bool", "byte", "complex64", "complex128", "error", "float32",
		"float64", "int", "int8", "int16", "int32", "int64", "rune", "string",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr", "true", "false",
		"nil", "iota", "len", "cap", "make", "new", "append", "copy", "delete",
	}

	return h.applyHighlighting(code, keywords, "//", "/*", "*/")
}

// highlightPython applies Python syntax highlighting
func (h *Highlighter) highlightPython(code string) string {
	keywords := []string{
		"False", "None", "True", "and", "as", "assert", "async", "await", "break",
		"class", "continue", "def", "del", "elif", "else", "except", "finally",
		"for", "from", "global", "if", "import", "in", "is", "lambda", "nonlocal",
		"not", "or", "pass", "raise", "return", "try", "while", "with", "yield",
		"int", "float", "str", "bool", "list", "dict", "tuple", "set",
	}

	return h.applyHighlighting(code, keywords, "#", `"""`, `"""`)
}

// highlightJavaScript applies JavaScript/TypeScript syntax highlighting
func (h *Highlighter) highlightJavaScript(code string) string {
	keywords := []string{
		"async", "await", "break", "case", "catch", "class", "const", "continue",
		"debugger", "default", "delete", "do", "else", "export", "extends", "finally",
		"for", "function", "if", "import", "in", "instanceof", "let", "new", "return",
		"super", "switch", "this", "throw", "try", "typeof", "var", "void", "while",
		"with", "yield", "boolean", "number", "string", "symbol", "undefined",
		"null", "true", "false", "Array", "Object", "String", "Number", "Boolean",
		"Math", "Date", "RegExp", "Promise", "console", "window", "document",
	}

	return h.applyHighlighting(code, keywords, "//", "/*", "*/")
}

// highlightJava applies Java syntax highlighting
func (h *Highlighter) highlightJava(code string) string {
	keywords := []string{
		"abstract", "assert", "boolean", "break", "byte", "case", "catch", "char",
		"class", "const", "continue", "default", "do", "double", "else", "enum",
		"extends", "final", "finally", "float", "for", "goto", "if", "implements",
		"import", "instanceof", "int", "interface", "long", "native", "new", "package",
		"private", "protected", "public", "return", "short", "static", "strictfp",
		"super", "switch", "synchronized", "this", "throw", "throws", "transient",
		"try", "void", "volatile", "while", "true", "false", "null",
	}

	return h.applyHighlighting(code, keywords, "//", "/*", "*/")
}

// highlightCpp applies C++ syntax highlighting
func (h *Highlighter) highlightCpp(code string) string {
	keywords := []string{
		"alignas", "alignof", "and", "and_eq", "asm", "auto", "bitand", "bitor",
		"bool", "break", "case", "catch", "char", "char16_t", "char32_t", "class",
		"compl", "const", "constexpr", "const_cast", "continue", "decltype", "default",
		"delete", "do", "double", "dynamic_cast", "else", "enum", "explicit", "export",
		"extern", "false", "float", "for", "friend", "goto", "if", "inline", "int",
		"long", "mutable", "namespace", "new", "noexcept", "not", "not_eq", "nullptr",
		"operator", "or", "or_eq", "private", "protected", "public", "register",
		"reinterpret_cast", "return", "short", "signed", "sizeof", "static",
		"static_assert", "static_cast", "struct", "switch", "template", "this",
		"thread_local", "throw", "true", "try", "typedef", "typeid", "typename",
		"union", "unsigned", "using", "virtual", "void", "volatile", "wchar_t",
		"while", "xor", "xor_eq", "#include", "#define", "#ifdef", "#ifndef", "#endif",
	}

	return h.applyHighlighting(code, keywords, "//", "/*", "*/")
}

// highlightRust applies Rust syntax highlighting
func (h *Highlighter) highlightRust(code string) string {
	keywords := []string{
		"as", "break", "const", "continue", "crate", "else", "enum", "extern",
		"false", "fn", "for", "if", "impl", "in", "let", "loop", "match", "mod",
		"move", "mut", "pub", "ref", "return", "self", "Self", "static", "struct",
		"super", "trait", "true", "type", "unsafe", "use", "where", "while",
		"bool", "char", "f32", "f64", "i8", "i16", "i32", "i64", "isize", "str",
		"u8", "u16", "u32", "u64", "usize", "String", "Vec", "HashMap", "Option",
		"Result", "Some", "None", "Ok", "Err", "println!", "vec!", "format!",
	}

	return h.applyHighlighting(code, keywords, "//", "/*", "*/")
}

// applyHighlighting applies syntax highlighting using regex patterns
func (h *Highlighter) applyHighlighting(code string, keywords []string, lineComment, blockCommentStart, blockCommentEnd string) string {
	lines := strings.Split(code, "\n")
	var highlightedLines []string

	for _, line := range lines {
		highlightedLine := h.highlightLine(line, keywords, lineComment, blockCommentStart, blockCommentEnd)
		highlightedLines = append(highlightedLines, highlightedLine)
	}

	return strings.Join(highlightedLines, "\n")
}

// highlightLine applies highlighting to a single line
func (h *Highlighter) highlightLine(line string, keywords []string, lineComment, blockCommentStart, blockCommentEnd string) string {
	// Handle comments first (they take precedence)
	if strings.Contains(line, lineComment) {
		parts := strings.SplitN(line, lineComment, 2)
		if len(parts) == 2 {
			return h.applyColor(parts[0], TokenNormal) + h.applyColor(lineComment+parts[1], TokenComment)
		}
	}

	// Simple keyword highlighting (basic implementation)
	result := line
	for _, keyword := range keywords {
		// Use word boundaries to avoid partial matches
		pattern := `\b` + regexp.QuoteMeta(keyword) + `\b`
		re := regexp.MustCompile(pattern)
		result = re.ReplaceAllStringFunc(result, func(match string) string {
			return h.applyColor(match, TokenKeyword)
		})
	}

	// Highlight strings (basic implementation)
	stringPattern := `"(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'|\x60(?:[^\x60\\]|\\.)*\x60`
	re := regexp.MustCompile(stringPattern)
	result = re.ReplaceAllStringFunc(result, func(match string) string {
		return h.applyColor(match, TokenString)
	})

	// Highlight numbers (basic implementation)
	numberPattern := `\b\d+(?:\.\d+)?\b`
	re = regexp.MustCompile(numberPattern)
	result = re.ReplaceAllStringFunc(result, func(match string) string {
		return h.applyColor(match, TokenNumber)
	})

	return result
}

// applyColor applies ANSI color codes based on token type
func (h *Highlighter) applyColor(text string, tokenType TokenType) string {
	// For now, return plain text - colors will be applied by the theme system
	// This is a basic implementation that can be extended with actual ANSI colors
	switch tokenType {
	case TokenKeyword:
		return text // Could add ANSI color codes here
	case TokenString:
		return text
	case TokenComment:
		return text
	case TokenNumber:
		return text
	case TokenOperator:
		return text
	case TokenBracket:
		return text
	case TokenTypeName:
		return text
	default:
		return text
	}
}
