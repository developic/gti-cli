package internal

import (
	"bufio"
	"math/rand"
	"strings"
	"sync"
	"time"

	"gti/src/assets"
)

var defaultWords = []string{
	"the", "quick", "brown", "fox", "jumps", "over", "lazy", "dog",
	"hello", "world", "typing", "speed", "test", "practice", "accuracy",
	"keyboard", "computer", "software", "development", "programming",
	"go", "language", "bubble", "tea", "terminal", "user", "interface",
}

var languageFiles = map[string]string{
	"english":    "eng",
	"spanish":    "spa",
	"french":     "fre",
	"german":     "ger",
	"japanese":   "jap",
	"russian":    "ru",
	"italian":    "ita",
	"portuguese": "por",
	"chinese":    "chi",
	"arabic":     "ara",
	"hindi":      "hin",
	"korean":     "kor",
	"dutch":      "dut",
	"swedish":    "swe",
	"czech":      "cze",
	"danish":     "dan",
	"finnish":    "fin",
	"greek":      "gre",
	"hebrew":     "heb",
	"hungarian":  "hun",
	"norwegian":  "nor",
	"polish":     "pol",
	"thai":       "tha",
	"turkish":    "tur",
	"random":     "ran",
}

var codeLanguageFiles = map[string]string{
	"go":         "go.snippets",
	"python":     "python.snippets",
	"javascript": "javascript.snippets",
	"java":       "java.snippets",
	"cpp":        "cpp.snippets",
	"rust":       "rust.snippets",
	"typescript": "typescript.snippets",
}

var loadedWords = make(map[string][]string)
var loadedCodeSnippets = make(map[string][]string)
var loadMutex sync.Mutex

func loadWords(language string) []string {
	loadMutex.Lock()
	defer loadMutex.Unlock()

	if words, exists := loadedWords[language]; exists {
		return words
	}

	fileName, exists := languageFiles[language]
	if !exists {
		fileName = languageFiles["random"]
	}

	filePath := "words/" + fileName

	data, err := assets.Words.ReadFile(filePath)
	if err != nil {
		return defaultWords
	}

	var words []string
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			words = append(words, line)
		}
	}

	if len(words) == 0 {
		words = defaultWords
	}

	loadedWords[language] = words
	return words
}

func GenerateWord(language string) string {
	rand.Seed(time.Now().UnixNano())
	words := loadWords(language)
	return words[rand.Intn(len(words))]
}

func GenerateWordsDynamic(count int, language string) string {
	rand.Seed(time.Now().UnixNano())
	var selected []string
	for i := 0; i < count; i++ {
		selected = append(selected, GenerateWord(language))
	}
	return strings.Join(selected, " ")
}

func IsLanguageSupported(language string) bool {
	_, exists := languageFiles[language]
	return exists
}

func loadCodeSnippets(language string) []string {
	loadMutex.Lock()
	defer loadMutex.Unlock()

	if snippets, exists := loadedCodeSnippets[language]; exists {
		return snippets
	}

	fileName, exists := codeLanguageFiles[language]
	if !exists {
		// Default to Go if language not found
		fileName = codeLanguageFiles["go"]
	}

	filePath := "code/" + fileName

	data, err := assets.Code.ReadFile(filePath)
	if err != nil {
		// Return a simple default code snippet
		return []string{"func main() {\n    fmt.Println(\"Hello, World!\")\n}"}
	}

	var snippets []string
	var currentSnippet strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(string(data)))

	for scanner.Scan() {
		line := scanner.Text()
		// Check for snippet separator (lines starting with #)
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			// Save previous snippet if it exists
			if currentSnippet.Len() > 0 {
				snippets = append(snippets, strings.TrimSuffix(currentSnippet.String(), "\n"))
				currentSnippet.Reset()
			}
		} else if strings.TrimSpace(line) != "" {
			// Add non-empty lines to current snippet
			currentSnippet.WriteString(line)
			currentSnippet.WriteString("\n")
		}
	}

	// Add the last snippet
	if currentSnippet.Len() > 0 {
		snippets = append(snippets, strings.TrimSuffix(currentSnippet.String(), "\n"))
	}

	if len(snippets) == 0 {
		snippets = []string{"func main() {\n    fmt.Println(\"Hello, World!\")\n}"}
	}

	loadedCodeSnippets[language] = snippets
	return snippets
}

func GenerateCodeSnippet(language string) string {
	rand.Seed(time.Now().UnixNano())
	snippets := loadCodeSnippets(language)
	return snippets[rand.Intn(len(snippets))]
}

func GenerateCodeSnippets(count int, language string) string {
	rand.Seed(time.Now().UnixNano())
	snippets := loadCodeSnippets(language)
	var selected []string

	for i := 0; i < count && i < len(snippets); i++ {
		selected = append(selected, snippets[rand.Intn(len(snippets))])
	}

	return strings.Join(selected, "\n\n")
}

func IsCodeLanguageSupported(language string) bool {
	_, exists := codeLanguageFiles[language]
	return exists
}
