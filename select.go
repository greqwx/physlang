// подбор формулы под искомую величину и вычисление ответа
package main

import (
	"errors"
	"fmt"
	"strings"
)

// solution это готовое решение для печати
// три строки правой колонки: формула, подстановка и результат
type solution struct {
	value       float64
	symbol      string
	formulaLine string
	substLine   string
	resultLine  string
}

// selectFormulas оставляет формулы, где левая часть совпадает с искомой
// и все зависимые величины уже есть в «дано»
func selectFormulas(reg []*formula, target string, known map[string]bool) []*formula {
	var out []*formula
	for _, f := range reg {
		if f.target != target {
			continue
		}
		ok := true
		for _, d := range f.deps {
			if !known[d] {
				ok = false
				break
			}
		}
		if ok {
			out = append(out, f)
		}
	}
	return out
}

// noFormulaError собирает ошибку со списком формул и недостающих величин
func noFormulaError(reg []*formula, target string, known map[string]bool) error {
	var cands []*formula
	for _, f := range reg {
		if f.target == target {
			cands = append(cands, f)
		}
	}
	if len(cands) == 0 {
		return fmt.Errorf("в реестре нет формулы для «%s»", target)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "нет формулы для «%s», не хватает величин:", target)
	for _, f := range cands {
		var miss []string
		for _, d := range f.deps {
			if !known[d] {
				miss = append(miss, d)
			}
		}
		fmt.Fprintf(&b, "\n  %s = %s  (нужно: %s)", f.target, f.rhs, strings.Join(miss, ", "))
	}
	return errors.New(b.String())
}

// solve переводит дано в СИ, считает по формуле и собирает строки решения
func solve(t *task, f *formula) (*solution, error) {
	// шкалы температуры различаются смещением, поэтому смешивать их нельзя
	tempText := ""
	tempFactor := 0.0
	for _, q := range t.given {
		if q.unit == nil || q.unit.d != (dim{4: 1}) {
			continue
		}
		if tempText == "" {
			tempText = q.text
			tempFactor = q.unit.f
			continue
		}
		if q.unit.f != tempFactor {
			return nil, fmt.Errorf("в задаче смешаны разные шкалы температуры: «%s» и «%s»", tempText, q.text)
		}
	}

	env := make(map[string]float64, len(t.given))
	uenv := make(map[string]*unit, len(t.given))
	disp := make(map[string]string, len(t.given))
	// значения переводятся в СИ один раз и дальше идут и в счёт, и в подстановку
	for _, q := range t.given {
		v := q.value
		if q.unit != nil {
			v *= q.unit.f
		}
		env[q.name] = v
		switch {
		case q.unit != nil:
			uenv[q.name] = siUnit(q.unit.d)
		case q.value == 0:
			uenv[q.name] = &unit{f: 1, any: true}
		default:
			uenv[q.name] = &unit{f: 1}
		}
		disp[q.name] = formatNumber(v)
	}

	v, err := f.expr.eval(env)
	if err != nil {
		return nil, err
	}

	// если размерность вывести не удалось, берётся подсказка из реестра
	u, uerr := f.expr.unitOf(uenv)
	if uerr != nil || u == nil {
		u = &unit{f: 1}
		if f.hint != nil {
			u = siUnit(f.hint.d)
		}
	} else if u.isZero() && f.hint != nil {
		u = siUnit(f.hint.d)
	}

	sym := u.symbol()
	result := f.target + " = " + formatNumber(v)
	if sym != "" {
		result += " " + sym
	}
	return &solution{
		value:       v,
		symbol:      sym,
		formulaLine: f.target + " = " + f.rhs,
		substLine:   f.target + " = " + substitute(f.rhs, disp),
		resultLine:  result,
	}, nil
}
