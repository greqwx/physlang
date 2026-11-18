// печать таблицы: столбцы «дано», «СИ» и «решение»
package main

import (
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"
)

// formatNumber печатает две значащие цыфры с запятой
// хвостовые нули убераются, чтобы 5 не стало 5,0
func formatNumber(v float64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	a := math.Abs(v)
	exp := int(math.Floor(math.Log10(a)))
	dec := 1 - exp
	if dec < 0 {
		dec = 0
	}
	if dec > 9 {
		dec = 9
	}
	s := strconv.FormatFloat(a, 'f', dec, 64)
	if strings.IndexByte(s, '.') >= 0 {
		s = strings.TrimRight(s, "0")
		s = strings.TrimSuffix(s, ".")
	}
	s = strings.ReplaceAll(s, ".", ",")
	if neg {
		s = "-" + s
	}
	return s
}

// exactValue проверяет, совпадает ли округлённое число с точным
// по этому выбирается знак = или ≈ в ответе
func exactValue(v float64, s string) bool {
	p, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", "."), 64)
	if err != nil {
		return false
	}
	return math.Abs(p-v) <= 1e-9*math.Max(1, math.Abs(v))
}

// tableRow это одна строка таблицы: три столбца и пометка разделителя
type tableRow struct {
	left  string
	si    string
	right string
	sep   bool
}

// renderTable собирает таблицу «дано, СИ, решение»
// ширина столбцов считается по рунам, иначе кириллица разъедется
func renderTable(w io.Writer, t *task, target string, s *solution) {
	right := []string{s.formulaLine, s.substLine, s.resultLine}
	n := len(t.given)
	if len(right) > n {
		n = len(right)
	}

	rows := []tableRow{{"Дано:", "СИ", "Решение", false}}
	// правая колонка может быть длиннее блока дано, тогда строки добиваются пустыми
	for i := 0; i < n; i++ {
		var r tableRow
		if i < len(t.given) {
			q := t.given[i]
			r.left = q.text
			if q.unit != nil && q.unit.f != 1 {
				r.si = formatNumber(q.value*q.unit.f) + " " + siUnit(q.unit.d).symbol()
			}
		}
		if i < len(right) {
			r.right = right[i]
		}
		rows = append(rows, r)
	}
	rows = append(rows, tableRow{sep: true})
	// строка искомого идёт после разделителя, как в тетраде
	rows = append(rows, tableRow{left: target + " - ?"})

	wl, ws, wr := 0, 0, 0
	for _, r := range rows {
		if r.sep {
			continue
		}
		wl = maxInt(wl, runeLen(r.left))
		ws = maxInt(ws, runeLen(r.si))
		wr = maxInt(wr, runeLen(r.right))
	}

	for _, r := range rows {
		if r.sep {
			fmt.Fprintf(w, "%s+%s+%s\n",
				strings.Repeat("-", wl+2),
				strings.Repeat("-", ws+2),
				strings.Repeat("-", wr+2))
			continue
		}
		line := " " + padRight(r.left, wl) + " | " + padRight(r.si, ws) + " | " + r.right
		fmt.Fprintln(w, strings.TrimRight(line, " "))
	}
}

// renderBlock печатает вариант целиком: заголовок, таблицу и ответ
func renderBlock(t *task, target string, s *solution, numbered bool, n int) string {
	var b strings.Builder
	if numbered {
		fmt.Fprintf(&b, "Вариант %d\n", n)
	}
	renderTable(&b, t, target, s)
	b.WriteString("\n")

	num := formatNumber(s.value)
	op := "≈"
	if exactValue(s.value, num) {
		op = "="
	}
	fmt.Fprintf(&b, "Ответ: %s %s %s", target, op, num)
	if s.symbol != "" {
		fmt.Fprintf(&b, " %s", s.symbol)
	}
	b.WriteString("\n")
	return b.String()
}

// runeLen считает символы, а не байты, это нужно для выравнивания таблицы
func runeLen(s string) int { return utf8.RuneCountInString(s) }

// padRight добивает строку пробелами до нужной ширины
func padRight(s string, n int) string {
	d := n - runeLen(s)
	if d <= 0 {
		return s
	}
	return s + strings.Repeat(" ", d)
}

// maxInt возвращает большее из двух чисел
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
