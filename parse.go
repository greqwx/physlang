// чтение файла задачи: разделы «дано» и «найти», строки величин
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// quantity это одна величина из раздела «дано»
// текст строки хранится как есть, чтобы печатать его в таблице
type quantity struct {
	name  string
	value float64
	unit  *unit
	text  string
	line  int
}

// task это разобранная задача
// given идёт в том же порядке, что и в файле, а byName нужен для поиска
type task struct {
	given  []*quantity
	byName map[string]*quantity
	target []string
}

// isIdentStart проверяет, что символ может начинать имя
func isIdentStart(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_'
}

// isIdentPart разрешает цифры внутри имени, так получается u0
func isIdentPart(c byte) bool {
	return isIdentStart(c) || c >= '0' && c <= '9'
}

// isIdent проверяет имя целиком
func isIdent(s string) bool {
	if s == "" || !isIdentStart(s[0]) {
		return false
	}
	for i := 1; i < len(s); i++ {
		if !isIdentPart(s[i]) {
			return false
		}
	}
	return true
}

// isRule отвечает, что строка состоит из одних дефисов
func isRule(s string) bool {
	for _, r := range s {
		switch r {
		case '-', '–', '—', '=', '_', ' ':
		default:
			return false
		}
	}
	return true
}

// normalizeHeader убирает скобки, дефисы и регистр
// так из «[----Дано----]» получается «дано»
func normalizeHeader(s string) string {
	r := strings.NewReplacer("[", "", "]", "", ":", "", "-", "", "–", "", "—", "", " ", "")
	return strings.ToLower(r.Replace(s))
}

// parseTask читает файл и раскладывает его на два раздела
func parseTask(path string) (*task, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	t := &task{byName: map[string]*quantity{}}
	section := ""
	// переводы строк приводятся к одному виду, чтобы разбор не зависел от системы
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	for i, raw := range lines {
		n := i + 1
		line := strings.TrimSpace(raw)
		// пустые строки и линии из дефисов не несут данных
		if line == "" || isRule(line) {
			continue
		}
		// заголовок в квадратных скобках задаёт, куда писать следующие строки
		if strings.HasPrefix(line, "[") {
			h := normalizeHeader(line)
			switch {
			case strings.Contains(h, "дано"):
				section = "given"
			case strings.Contains(h, "найти"):
				section = "target"
			default:
				return nil, fmt.Errorf("%s:%d: неизвестный раздел «%s»", path, n, line)
			}
			continue
		}
		switch section {
		case "given":
			if err := t.addGiven(path, n, line); err != nil {
				return nil, err
			}
		case "target":
			if err := t.addTarget(path, n, line); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("%s:%d: строка вне разделов «дано» и «найти»", path, n)
		}
	}
	if len(t.given) == 0 {
		return nil, fmt.Errorf("%s: раздел «дано» пуст", path)
	}
	if len(t.target) == 0 {
		return nil, fmt.Errorf("%s: раздел «найти» пуст", path)
	}
	return t, nil
}

// addGiven разбирает строку «имя = число единица»
// единица необязательна, u = 0 значит ноль без размерности
func (t *task) addGiven(path string, n int, line string) error {
	eq := strings.Index(line, "=")
	if eq < 0 {
		return fmt.Errorf("%s:%d: ожидается «имя = значение единица»", path, n)
	}
	name := strings.TrimSpace(line[:eq])
	if !isIdent(name) {
		return fmt.Errorf("%s:%d: плохое имя величины «%s»", path, n, name)
	}
	if _, dup := t.byName[name]; dup {
		return fmt.Errorf("%s:%d: величина «%s» уже задана", path, n, name)
	}
	rest := strings.TrimSpace(line[eq+1:])
	valStr := rest
	unitStr := ""
	if i := strings.IndexAny(rest, " \t"); i >= 0 {
		valStr = strings.TrimSpace(rest[:i])
		unitStr = strings.TrimSpace(rest[i:])
	}
	// число может быть записано с запятой, единица занимает всё после пробела
	v, err := strconv.ParseFloat(strings.ReplaceAll(valStr, ",", "."), 64)
	if err != nil {
		return fmt.Errorf("%s:%d: не разобрать число «%s»", path, n, valStr)
	}
	q := &quantity{name: name, value: v, text: line, line: n}
	if unitStr != "" {
		u, err := parseUnit(unitStr)
		if err != nil {
			return fmt.Errorf("%s:%d: %v", path, n, err)
		}
		q.unit = u
	}
	t.given = append(t.given, q)
	t.byName[name] = q
	return nil
}

// addTarget разбирает строку «имя - ?»
// знаки вопроса и дефисы отбрасываются, остаётся имя
func (t *task) addTarget(path string, n int, line string) error {
	s := strings.TrimSuffix(strings.TrimSpace(line), "?")
	s = strings.TrimRight(s, " -=:")
	s = strings.TrimSpace(s)
	if !isIdent(s) {
		return fmt.Errorf("%s:%d: ожидается «имя - ?», получено «%s»", path, n, line)
	}
	for _, x := range t.target {
		if x == s {
			return fmt.Errorf("%s:%d: «%s» уже в списке искомого", path, n, s)
		}
	}
	t.target = append(t.target, s)
	return nil
}
