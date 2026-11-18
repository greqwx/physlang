// реестр формул: чтение файла, где одна строка это одна формула
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// formula это одна строка реестра
// target это левая часть, rhs текст правой части, expr разобранное дерево
// deps это имена из правой части, hint необязательная единица ответа
type formula struct {
	target string
	rhs    string
	expr   *expr
	deps   []string
	hint   *unit
	line   int
}

// parseRegistry разбирает реестр построчно
// пустые строки и строки с # пропускаются
// после ; можно указать единицу ответа, если её не вывести из размерностей
func parseRegistry(text string) ([]*formula, error) {
	var out []*formula
	sc := bufio.NewScanner(strings.NewReader(text))
	n := 0
	for sc.Scan() {
		n++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		if line == "" {
			continue
		}
		var hint *unit
		if i := strings.IndexByte(line, ';'); i >= 0 {
			us := strings.TrimSpace(line[i+1:])
			line = strings.TrimSpace(line[:i])
			if us != "" {
				u, err := parseUnit(us)
				if err != nil {
					return nil, fmt.Errorf("реестр, строка %d: %v", n, err)
				}
				hint = u
			}
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			return nil, fmt.Errorf("реестр, строка %d: нет знака «=»", n)
		}
		target := strings.TrimSpace(line[:eq])
		rhs := strings.TrimSpace(line[eq+1:])
		if !isIdent(target) {
			return nil, fmt.Errorf("реестр, строка %d: плохое имя «%s»", n, target)
		}
		e, err := parseExpr(rhs)
		if err != nil {
			return nil, fmt.Errorf("реестр, строка %d: %v", n, err)
		}
		set := map[string]bool{}
		e.collect(set)
		delete(set, target)
		out = append(out, &formula{
			target: target,
			rhs:    rhs,
			expr:   e,
			deps:   sortedKeys(set),
			hint:   hint,
			line:   n,
		})
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("реестр формул пуст")
	}
	return out, nil
}

// sortedKeys отдаёт имена по алфавиту, чтобы порядок в ошибках был устойчив
func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// loadRegistry берёт формулы из -f, потом из файла рядом с программой
// и только если ничего не нашлось, использует встроенный набор
func loadRegistry(path string) ([]*formula, error) {
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		return parseRegistry(string(data))
	}
	if exe, err := os.Executable(); err == nil {
		p := filepath.Join(filepath.Dir(exe), "formulas.txt")
		if data, err := os.ReadFile(p); err == nil {
			return parseRegistry(string(data))
		}
	}
	return parseRegistry(embeddedFormulas)
}
