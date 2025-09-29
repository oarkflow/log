package log

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"
	"unsafe"
)

// FormatterArgs is a parsed struct from json input
type FormatterArgs struct {
	Time       string // "2019-07-10T05:35:54.277Z"
	Level      string // "info"
	Caller     string // "prog.go:42"
	CallerFunc string // "main.main"
	Goid       string // "123"
	Stack      string // "<stack string>"
	Message    string // "a structure message"
	Category   string // "cat1"
	KeyValues  []struct {
		Key       string // "foo"
		Value     string // "bar"
		ValueType byte   // 's'
	}
}

// Get gets the value associated with the given key.
func (args *FormatterArgs) Get(key string) (value string) {
	for i := len(args.KeyValues) - 1; i >= 0; i-- {
		kv := &args.KeyValues[i]
		if kv.Key == key {
			value = kv.Value
			break
		}
	}
	return
}

func formatterArgsPos(key string) (pos int) {
	switch key {
	case "time":
		pos = 1
	case "level":
		pos = 2
	case "caller":
		pos = 3
	case "callerfunc":
		pos = 4
	case "goid":
		pos = 5
	case "stack":
		pos = 6
	case "message", "msg":
		pos = 7
	case "category":
		pos = 8
	}
	return
}

// parseFormatterArgs extracts json string to json items
func parseFormatterArgs(json []byte, args *FormatterArgs) {
	// treat formatter args as []string
	const size = int(unsafe.Sizeof(FormatterArgs{}) / unsafe.Sizeof(""))
	//nolint:all
	slice := *(*[]string)(unsafe.Pointer(&reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(args)), Len: size, Cap: size,
	}))
	var keys = true
	var key, str []byte
	var ok bool
	var typ byte
	_ = json[len(json)-1] // remove bounds check
	if json[0] != '{' {
		return
	}
	for i := 1; i < len(json); i++ {
		if keys {
			if json[i] != '"' {
				continue
			}
			i, str, _, ok = jsonParseString(json, i+1)
			if !ok {
				return
			}
			key = str[1 : len(str)-1]
		}
		for ; i < len(json); i++ {
			if json[i] <= ' ' || json[i] == ',' || json[i] == ':' {
				continue
			}
			break
		}
		i, typ, str, ok = jsonParseAny(json, i, true)
		if !ok {
			return
		}
		switch typ {
		case 's':
			str = str[1 : len(str)-1]
		case 'S':
			str = jsonUnescape(str[1:len(str)-1], str[:0])
			typ = 's'
		}
		pos := formatterArgsPos(b2s(key))
		if pos == 0 && args.Time == "" {
			pos = 1
		}
		if pos != 0 {
			if pos == 2 && len(str) != 0 && str[len(str)-1] == '\n' {
				str = str[:len(str)-1]
			}
			if slice[pos-1] == "" {
				slice[pos-1] = b2s(str)
			}
		} else {
			args.KeyValues = append(args.KeyValues, struct {
				Key, Value string
				ValueType  byte
			}{b2s(key), b2s(str), typ})
		}
	}

	if args.Level == "" {
		args.Level = "????"
	}
}

func jsonParseString(json []byte, i int) (int, []byte, bool, bool) {
	var s = i
	_ = json[len(json)-1] // remove bounds check
	for ; i < len(json); i++ {
		if json[i] > '\\' {
			continue
		}
		if json[i] == '"' {
			return i + 1, json[s-1 : i+1], false, true
		}
		if json[i] == '\\' {
			i++
			for ; i < len(json); i++ {
				if json[i] > '\\' {
					continue
				}
				if json[i] == '"' {
					// look for an escaped slash
					if json[i-1] == '\\' {
						n := 0
						for j := i - 2; j > 0; j-- {
							if json[j] != '\\' {
								break
							}
							n++
						}
						if n%2 == 0 {
							continue
						}
					}
					return i + 1, json[s-1 : i+1], true, true
				}
			}
			break
		}
	}
	return i, json[s-1:], false, false
}

// jsonParseAny parses the next value from a json string.
// A Result is returned when the hit param is set.
// The return values are (i int, res Result, ok bool)
func jsonParseAny(json []byte, i int, hit bool) (int, byte, []byte, bool) {
	var typ byte
	var val []byte
	_ = json[len(json)-1] // remove bounds check
	for ; i < len(json); i++ {
		if json[i] == '{' || json[i] == '[' {
			i, val = jsonParseSquash(json, i)
			if hit {
				typ = 'o'
			}
			return i, typ, val, true
		}
		if json[i] <= ' ' {
			continue
		}
		switch json[i] {
		case '"':
			i++
			var vesc bool
			var ok bool
			i, val, vesc, ok = jsonParseString(json, i)
			typ = 's'
			if !ok {
				return i, typ, val, false
			}
			if hit && vesc {
				typ = 'S'
			}
			return i, typ, val, true
		case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			i, val = jsonParseNumber(json, i)
			if hit {
				typ = 'n'
			}
			return i, typ, val, true
		case 't', 'f', 'n':
			vc := json[i]
			i, val = jsonParseLiteral(json, i)
			if hit {
				switch vc {
				case 't':
					typ = 't'
				case 'f':
					typ = 'f'
				}
				return i, typ, val, true
			}
		}
	}
	return i, typ, val, false
}

func jsonParseSquash(json []byte, i int) (int, []byte) {
	// expects that the lead character is a '[' or '{' or '('
	// squash the value, ignoring all nested arrays and objects.
	// the first '[' or '{' or '(' has already been read
	s := i
	i++
	depth := 1
	_ = json[len(json)-1] // remove bounds check
	for ; i < len(json); i++ {
		if json[i] >= '"' && json[i] <= '}' {
			switch json[i] {
			case '"':
				i++
				s2 := i
				for ; i < len(json); i++ {
					if json[i] > '\\' {
						continue
					}
					if json[i] == '"' {
						// look for an escaped slash
						if json[i-1] == '\\' {
							n := 0
							for j := i - 2; j > s2-1; j-- {
								if json[j] != '\\' {
									break
								}
								n++
							}
							if n%2 == 0 {
								continue
							}
						}
						break
					}
				}
			case '{', '[', '(':
				depth++
			case '}', ']', ')':
				depth--
				if depth == 0 {
					i++
					return i, json[s:i]
				}
			}
		}
	}
	return i, json[s:]
}

func jsonParseNumber(json []byte, i int) (int, []byte) {
	var s = i
	i++
	_ = json[len(json)-1] // remove bounds check
	for ; i < len(json); i++ {
		if json[i] <= ' ' || json[i] == ',' || json[i] == ']' ||
			json[i] == '}' {
			return i, json[s:i]
		}
	}
	return i, json[s:]
}

func jsonParseLiteral(json []byte, i int) (int, []byte) {
	var s = i
	i++
	_ = json[len(json)-1] // remove bounds check
	for ; i < len(json); i++ {
		if json[i] < 'a' || json[i] > 'z' {
			return i, json[s:i]
		}
	}
	return i, json[s:]
}

// jsonUnescape unescapes a string
func jsonUnescape(json, str []byte) []byte {
	_ = json[len(json)-1] // remove bounds check
	var p [6]byte
	for i := 0; i < len(json); i++ {
		switch {
		default:
			str = append(str, json[i])
		case json[i] < ' ':
			return str
		case json[i] == '\\':
			i++
			if i >= len(json) {
				return str
			}
			switch json[i] {
			default:
				return str
			case '\\':
				str = append(str, '\\')
			case '/':
				str = append(str, '/')
			case 'b':
				str = append(str, '\b')
			case 'f':
				str = append(str, '\f')
			case 'n':
				str = append(str, '\n')
			case 'r':
				str = append(str, '\r')
			case 't':
				str = append(str, '\t')
			case '"':
				str = append(str, '"')
			case 'u':
				if i+5 > len(json) {
					return str
				}
				m, _ := strconv.ParseUint(b2s(json[i+1:i+5]), 16, 64)
				r := rune(m)
				i += 5
				if utf16.IsSurrogate(r) {
					// need another code
					if len(json[i:]) >= 6 && json[i] == '\\' &&
						json[i+1] == 'u' {
						// we expect it to be correct so just consume it
						m, _ = strconv.ParseUint(b2s(json[i+2:i+6]), 16, 64)
						r = utf16.DecodeRune(r, rune(m))
						i += 6
					}
				}
				str = append(str, p[:utf8.EncodeRune(p[:], r)]...)
				i-- // backtrack index by one
			}
		}
	}
	return str
}

// Formatter defines an interface for formatting log entries
type Formatter interface {
	Format(entry *Entry) ([]byte, error)
}

// JSONFormatter formats entries as JSON
type JSONFormatter struct{}

// Format formats the entry as JSON
func (f *JSONFormatter) Format(entry *Entry) ([]byte, error) {
	// For JSON formatting, we can simply return the original buffer
	// since it already contains valid JSON
	if len(entry.buf) > 0 {
		return entry.buf, nil
	}

	// Fallback: create a minimal JSON entry if buffer is empty
	var buf bytes.Buffer
	buf.WriteString(`{"time":"`)
	buf.WriteString(time.Now().Format("2006-01-02T15:04:05.999Z07:00"))
	buf.WriteString(`","level":"`)
	switch entry.Level {
	case DebugLevel:
		buf.WriteString("debug")
	case InfoLevel:
		buf.WriteString("info")
	case WarnLevel:
		buf.WriteString("warn")
	case ErrorLevel:
		buf.WriteString("error")
	case TraceLevel:
		buf.WriteString("trace")
	case FatalLevel:
		buf.WriteString("fatal")
	case PanicLevel:
		buf.WriteString("panic")
	default:
		buf.WriteString("????")
	}
	buf.WriteString("\"}\n")
	return buf.Bytes(), nil
}

// HumanReadableFormatter formats entries in a human-readable format
type HumanReadableFormatter struct {
	ShowTimestamp bool
	ShowLevel     bool
	ShowCaller    bool
}

// Format formats the entry in a human-readable way
func (f *HumanReadableFormatter) Format(entry *Entry) ([]byte, error) {
	var buf bytes.Buffer

	// Add timestamp
	if f.ShowTimestamp {
		now := time.Now()
		if entry.logger != nil && entry.logger.TimeLocation != nil {
			buf.WriteString(now.In(entry.logger.TimeLocation).Format("2006-01-02 15:04:05"))
		} else {
			buf.WriteString(now.Format("2006-01-02 15:04:05"))
		}
		buf.WriteByte(' ')
	}

	// Add level
	if f.ShowLevel {
		switch entry.Level {
		case DebugLevel:
			buf.WriteString("[DEBUG] ")
		case InfoLevel:
			buf.WriteString("[INFO]  ")
		case WarnLevel:
			buf.WriteString("[WARN]  ")
		case ErrorLevel:
			buf.WriteString("[ERROR] ")
		case TraceLevel:
			buf.WriteString("[TRACE] ")
		case FatalLevel:
			buf.WriteString("[FATAL] ")
		case PanicLevel:
			buf.WriteString("[PANIC] ")
		default:
			buf.WriteString("[????]  ")
		}
	}

	// Add caller information if available
	if f.ShowCaller && entry.logger != nil && entry.logger.Caller > 0 {
		// This would need to be extracted from the existing buffer
		// For now, we'll skip this as it requires more complex parsing
	}

	// Add message
	if len(entry.buf) > 0 {
		// Extract message from JSON buffer (simplified)
		msg := extractMessageFromBuffer(entry.buf)
		buf.WriteString(msg)
	}

	// Add newline only if the last character is not already a newline
	result := buf.Bytes()
	if len(result) == 0 || result[len(result)-1] != '\n' {
		buf.WriteByte('\n')
	}

	return buf.Bytes(), nil
}

// extractMessageFromBuffer extracts the message field from a JSON buffer
func extractMessageFromBuffer(buf []byte) string {
	jsonStr := string(buf)
	if strings.Contains(jsonStr, `"message":`) {
		// Find the message field
		msgStart := strings.Index(jsonStr, `"message":`) + 10
		if msgStart > 10 {
			// Find the end of the message value
			remaining := jsonStr[msgStart:]
			if strings.HasPrefix(remaining, `"`) {
				// Quoted message
				msgEnd := strings.Index(remaining[1:], `"`) + 1
				if msgEnd > 0 {
					return remaining[1:msgEnd]
				}
			}
		}
	}
	return jsonStr
}

// DefaultJSONFormatter is the default JSON formatter instance
var DefaultJSONFormatter = &JSONFormatter{}

// DefaultTemplateFormatter creates a template formatter with the given pattern
func DefaultTemplateFormatter(pattern string) *TemplateFormatter {
	return &TemplateFormatter{Pattern: pattern}
}

// TemplateFormatter formats entries using a template pattern with conditional support
type TemplateFormatter struct {
	Pattern string
}

// Format formats the entry using the template pattern
func (f *TemplateFormatter) Format(entry *Entry) ([]byte, error) {
	var buf bytes.Buffer
	pattern := f.Pattern
	i := 0

	for i < len(pattern) {
		if pattern[i] == '{' && i+1 < len(pattern) && pattern[i+1] == '{' {
			i += 2
			start := i

			// Check for conditional syntax
			if i+3 < len(pattern) && pattern[i:i+3] == "if " {
				i += 3
				condStart := i

				// Find the end of the condition
				for i < len(pattern) && !(pattern[i] == '}' && i+1 < len(pattern) && pattern[i+1] == '}') {
					i++
				}

				if i >= len(pattern) {
					return nil, fmt.Errorf("unclosed conditional statement")
				}

				condition := pattern[condStart:i]
				i += 2 // Skip }}

				// Parse condition and content
				content, endPos := f.parseConditional(pattern, i, condition, entry)
				buf.WriteString(content)
				i = endPos
			} else {
				// Regular template variable
				for i < len(pattern) && !(pattern[i] == '}' && i+1 < len(pattern) && pattern[i+1] == '}') {
					i++
				}

				if i >= len(pattern) {
					return nil, fmt.Errorf("unclosed template variable")
				}

				varName := pattern[start:i]
				i += 2 // Skip }}

				value := f.getTemplateValue(varName, entry)
				buf.WriteString(value)
			}
		} else {
			buf.WriteByte(pattern[i])
			i++
		}
	}

	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

// parseConditional parses conditional statements like {{if err}}...{{end}}
func (f *TemplateFormatter) parseConditional(pattern string, start int, condition string, entry *Entry) (string, int) {
	var buf bytes.Buffer
	i := start

	// Find the {{end}}
	for i < len(pattern) {
		if pattern[i] == '{' && i+1 < len(pattern) && pattern[i+1] == '{' &&
			i+2 < len(pattern) && pattern[i+2] == 'e' && i+3 < len(pattern) && pattern[i+3] == 'n' &&
			i+4 < len(pattern) && pattern[i+4] == 'd' && i+5 < len(pattern) && pattern[i+5] == '}' &&
			i+6 < len(pattern) && pattern[i+6] == '}' {
			// Found {{end}}
			i += 7
			break
		}

		if pattern[i] == '{' && i+1 < len(pattern) && pattern[i+1] == '{' {
			// Nested template variable
			i += 2
			nestedStart := i
			for i < len(pattern) && !(pattern[i] == '}' && i+1 < len(pattern) && pattern[i+1] == '}') {
				i++
			}
			if i < len(pattern) {
				varName := pattern[nestedStart:i]
				i += 2
				value := f.getTemplateValue(varName, entry)
				buf.WriteString(value)
			}
		} else {
			buf.WriteByte(pattern[i])
			i++
		}
	}

	// Check condition
	shouldInclude := f.evaluateCondition(condition, entry)

	if shouldInclude {
		return buf.String(), i
	}
	return "", i
}

// evaluateCondition evaluates a conditional expression
func (f *TemplateFormatter) evaluateCondition(condition string, entry *Entry) bool {
	switch condition {
	case "err", "error":
		return f.hasError(entry)
	default:
		return false
	}
}

// hasError checks if the entry has a non-empty error field
func (f *TemplateFormatter) hasError(entry *Entry) bool {
	if len(entry.buf) == 0 {
		return false
	}

	jsonStr := string(entry.buf)
	// Find the error field in JSON
	errorPattern := `"error":`
	errorIndex := strings.Index(jsonStr, errorPattern)
	if errorIndex == -1 {
		return false
	}

	// Look for the value after "error":
	valueStart := errorIndex + len(errorPattern)

	// Skip whitespace
	for valueStart < len(jsonStr) && (jsonStr[valueStart] == ' ' || jsonStr[valueStart] == '\t') {
		valueStart++
	}

	if valueStart >= len(jsonStr) {
		return false
	}

	// Check the value type and content
	switch jsonStr[valueStart] {
	case '"':
		// String value - find the end quote
		valueStart++ // Skip opening quote
		endQuote := strings.Index(jsonStr[valueStart:], `"`)
		if endQuote == -1 {
			return false
		}
		errorValue := jsonStr[valueStart : valueStart+endQuote]
		return errorValue != ""
	case 'n':
		// null value
		return !strings.HasPrefix(jsonStr[valueStart:], "null")
	case 't', 'f':
		// boolean value - true means there is an error
		return strings.HasPrefix(jsonStr[valueStart:], "true")
	default:
		// Number or other value - consider non-zero as error
		return true
	}
}

// extractErrorValue extracts the error value from JSON buffer
func (f *TemplateFormatter) extractErrorValue(buf []byte) string {
	if len(buf) == 0 {
		return ""
	}

	jsonStr := string(buf)
	// Find the error field in JSON
	errorPattern := `"error":`
	errorIndex := strings.Index(jsonStr, errorPattern)
	if errorIndex == -1 {
		return ""
	}

	// Look for the value after "error":
	valueStart := errorIndex + len(errorPattern)

	// Skip whitespace
	for valueStart < len(jsonStr) && (jsonStr[valueStart] == ' ' || jsonStr[valueStart] == '\t') {
		valueStart++
	}

	if valueStart >= len(jsonStr) {
		return ""
	}

	// Extract the value based on its type
	switch jsonStr[valueStart] {
	case '"':
		// String value - find the end quote
		valueStart++ // Skip opening quote
		endQuote := strings.Index(jsonStr[valueStart:], `"`)
		if endQuote == -1 {
			return ""
		}
		return jsonStr[valueStart : valueStart+endQuote]
	case 'n':
		// null value
		if strings.HasPrefix(jsonStr[valueStart:], "null") {
			return ""
		}
		return "null"
	case 't':
		// boolean true
		if strings.HasPrefix(jsonStr[valueStart:], "true") {
			return "true"
		}
		return ""
	case 'f':
		// boolean false
		if strings.HasPrefix(jsonStr[valueStart:], "false") {
			return "false"
		}
		return ""
	default:
		// Number or other value - extract until next comma, whitespace, or end
		endPos := valueStart
		for endPos < len(jsonStr) && jsonStr[endPos] != ',' && jsonStr[endPos] != ' ' &&
			jsonStr[endPos] != '\t' && jsonStr[endPos] != '\n' && jsonStr[endPos] != '\r' {
			// Handle nested objects/arrays (simple case)
			if jsonStr[endPos] == '{' || jsonStr[endPos] == '[' {
				// For simplicity, just take until the next comma
				break
			}
			endPos++
		}
		if endPos > valueStart {
			return jsonStr[valueStart:endPos]
		}
		return ""
	}
}

// getTemplateValue returns the value for a template variable
func (f *TemplateFormatter) getTemplateValue(varName string, entry *Entry) string {
	switch varName {
	case "msg", "message":
		// Extract message from the JSON buffer
		return extractMessageFromBuffer(entry.buf)
	case "level":
		switch entry.Level {
		case DebugLevel:
			return "DEBUG"
		case InfoLevel:
			return "INFO"
		case WarnLevel:
			return "WARN"
		case ErrorLevel:
			return "ERROR"
		case TraceLevel:
			return "TRACE"
		case FatalLevel:
			return "FATAL"
		case PanicLevel:
			return "PANIC"
		default:
			return "????"
		}
	case "time":
		now := time.Now()
		if entry.logger != nil && entry.logger.TimeLocation != nil {
			return now.In(entry.logger.TimeLocation).Format("2006-01-02 15:04:05")
		}
		return now.Format("2006-01-02 15:04:05")
	case "caller":
		// Extract caller from the JSON buffer
		jsonStr := string(entry.buf)
		if strings.Contains(jsonStr, `"caller":`) {
			// Simple extraction - find caller field
			callerStart := strings.Index(jsonStr, `"caller":`) + 10
			if callerStart > 10 {
				remaining := jsonStr[callerStart:]
				if strings.HasPrefix(remaining, `"`) {
					callerEnd := strings.Index(remaining[1:], `"`) + 1
					if callerEnd > 0 {
						return remaining[1:callerEnd]
					}
				}
			}
		}
		return ""
	case "err", "error":
		// Extract error from the JSON buffer using improved parsing
		return f.extractErrorValue(entry.buf)
	default:
		return ""
	}
}

// DefaultHumanReadableFormatter is the default human-readable formatter instance
var DefaultHumanReadableFormatter = &HumanReadableFormatter{
	ShowTimestamp: true,
	ShowLevel:     true,
	ShowCaller:    false,
}
