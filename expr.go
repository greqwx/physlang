// разбор формул, вычисление значений и вывод размерности
// делено на два прохода: значение в СИ и размерность ответа
package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// expr это узел дерева выражения
// op 0 значит лист: число в num или имя величины в name
// остальные op повторяют знаки действий, 'm' это унарный минус
type expr struct {
	op   byte
	num  float64
	name string
	l    *expr
	r    *expr
}

// parseExpr разбирает формулу целиком и следит, чтобы лишних символов не осталось
func parseExpr(s string) (*expr, error) {
	p := &exprParser{s: s}
	e, err := p.parseSum()
	if err != nil {
		return nil, err
	}
	p.skipSpace()
	if p.pos < len(p.s) {
		return nil, fmt.Errorf("лишние символы «%s»", p.s[p.pos:])
	}
	return e, nil
}

// exprParser хранит строку и текущую позицию разбора
type exprParser struct {
	s   string
	pos int
}

// skipSpace пропускает пробелы и табуляции
func (p *exprParser) skipSpace() {
	for p.pos < len(p.s) && (p.s[p.pos] == ' ' || p.s[p.pos] == '\t') {
		p.pos++
	}
}

// peek смотрит текущий символ, ноль значит конец строки
func (p *exprParser) peek() byte {
	p.skipSpace()
	if p.pos >= len(p.s) {
		return 0
	}
	return p.s[p.pos]
}

// parseSum разбирает сложение и вычитание слева направо
func (p *exprParser) parseSum() (*expr, error) {
	left, err := p.parseProduct()
	if err != nil {
		return nil, err
	}
	for {
		c := p.peek()
		if c != '+' && c != '-' {
			return left, nil
		}
		p.pos++
		right, err := p.parseProduct()
		if err != nil {
			return nil, err
		}
		left = &expr{op: c, l: left, r: right}
	}
}

// parseProduct разбирает умножение и деление
func (p *exprParser) parseProduct() (*expr, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for {
		c := p.peek()
		if c != '*' && c != '/' {
			return left, nil
		}
		p.pos++
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = &expr{op: c, l: left, r: right}
	}
}

// parseUnary разбирает унарный минус и записывает его узлом 'm'
func (p *exprParser) parseUnary() (*expr, error) {
	switch p.peek() {
	case '-':
		p.pos++
		e, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &expr{op: 'm', l: e}, nil
	case '+':
		p.pos++
		return p.parseUnary()
	}
	return p.parsePower()
}

// parsePower разбирает степень, она правоассоциативна и пускает отрицательный показатель
func (p *exprParser) parsePower() (*expr, error) {
	base, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	if p.peek() != '^' {
		return base, nil
	}
	p.pos++
	exp, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	return &expr{op: '^', l: base, r: exp}, nil
}

// parsePrimary разбирает число, имя величины или скобки
func (p *exprParser) parsePrimary() (*expr, error) {
	c := p.peek()
	switch {
	case c == 0:
		return nil, fmt.Errorf("неожиданный конец формулы")
	case c == '(':
		p.pos++
		e, err := p.parseSum()
		if err != nil {
			return nil, err
		}
		if p.peek() != ')' {
			return nil, fmt.Errorf("не закрыта скобка")
		}
		p.pos++
		return e, nil
	case c >= '0' && c <= '9', c == '.':
		start := p.pos
		for p.pos < len(p.s) {
			ch := p.s[p.pos]
			if ch >= '0' && ch <= '9' || ch == '.' || ch == ',' {
				p.pos++
				continue
			}
			break
		}
		// запятая в числе заменяется на точку, чтобы принять школьную запись
		txt := strings.ReplaceAll(p.s[start:p.pos], ",", ".")
		v, err := strconv.ParseFloat(txt, 64)
		if err != nil {
			return nil, fmt.Errorf("плохое число «%s»", txt)
		}
		return &expr{num: v}, nil
	case isIdentStart(c):
		start := p.pos
		for p.pos < len(p.s) && isIdentPart(p.s[p.pos]) {
			p.pos++
		}
		return &expr{name: p.s[start:p.pos]}, nil
	}
	return nil, fmt.Errorf("неожиданный символ «%c»", c)
}

// eval считает значение по величинам, уже переведённым в СИ
func (e *expr) eval(env map[string]float64) (float64, error) {
	switch e.op {
	case 0:
		if e.name == "" {
			return e.num, nil
		}
		v, ok := env[e.name]
		if !ok {
			return 0, fmt.Errorf("нет значения для «%s»", e.name)
		}
		return v, nil
	case 'm':
		v, err := e.l.eval(env)
		return -v, err
	case '+', '-', '*', '/':
		a, err := e.l.eval(env)
		if err != nil {
			return 0, err
		}
		b, err := e.r.eval(env)
		if err != nil {
			return 0, err
		}
		switch e.op {
		case '+':
			return a + b, nil
		case '-':
			return a - b, nil
		case '*':
			return a * b, nil
		default:
			if b == 0 {
				return 0, fmt.Errorf("деление на ноль")
			}
			return a / b, nil
		}
	case '^':
		a, err := e.l.eval(env)
		if err != nil {
			return 0, err
		}
		b, err := e.r.eval(env)
		if err != nil {
			return 0, err
		}
		return math.Pow(a, b), nil
	}
	return 0, fmt.Errorf("неизвестная операция")
}

// unitOf выводит размерность результата из размерностей величин
// сложение требует одинаковых размерностей, ноль без единицы подчиняется соседу
func (e *expr) unitOf(env map[string]*unit) (*unit, error) {
	switch e.op {
	case 0:
		if e.name == "" {
			return &unit{f: 1}, nil
		}
		u, ok := env[e.name]
		if !ok {
			return &unit{f: 1}, nil
		}
		return u, nil
	case 'm':
		return e.l.unitOf(env)
	case '+', '-':
		a, err := e.l.unitOf(env)
		if err != nil {
			return nil, err
		}
		b, err := e.r.unitOf(env)
		if err != nil {
			return nil, err
		}
		if a.any && b.any {
			return &unit{f: 1, any: true}, nil
		}
		if a.any {
			return siUnit(b.d), nil
		}
		if b.any {
			return siUnit(a.d), nil
		}
		if a.equalDim(b) {
			return siUnit(a.d), nil
		}
		return nil, fmt.Errorf("складываются разные величины: %s и %s", a.symbol(), b.symbol())
	case '*':
		a, err := e.l.unitOf(env)
		if err != nil {
			return nil, err
		}
		b, err := e.r.unitOf(env)
		if err != nil {
			return nil, err
		}
		return a.mul(b), nil
	case '/':
		a, err := e.l.unitOf(env)
		if err != nil {
			return nil, err
		}
		b, err := e.r.unitOf(env)
		if err != nil {
			return nil, err
		}
		return a.div(b), nil
	case '^':
		// степень в размерности должна быть целой, иначе единицу не вывести
		v, err := e.r.eval(map[string]float64{})
		if err != nil {
			return nil, fmt.Errorf("степень должна быть числом")
		}
		n := int(math.Round(v))
		if math.Abs(v-float64(n)) > 1e-9 {
			return nil, fmt.Errorf("степень должна быть целой")
		}
		a, err := e.l.unitOf(env)
		if err != nil {
			return nil, err
		}
		return a.pow(n), nil
	}
	return &unit{f: 1}, nil
}

// collect собирает имена, от которых зависит формула
// по этому списку проверяется, хватает ли данных в разделе «дано»
func (e *expr) collect(out map[string]bool) {
	if e == nil {
		return
	}
	if e.op == 0 {
		if e.name != "" {
			out[e.name] = true
		}
		return
	}
	e.l.collect(out)
	e.r.collect(out)
}

// substitute подставляет числа вместо имён прямо в текст формулы
// так в решении остаётся исходная запись, а не пересобранная заново
func substitute(src string, vals map[string]string) string {
	var b strings.Builder
	for i := 0; i < len(src); {
		if isIdentStart(src[i]) {
			j := i
			for j < len(src) && isIdentPart(src[j]) {
				j++
			}
			name := src[i:j]
			if v, ok := vals[name]; ok {
				b.WriteString(v)
			} else {
				b.WriteString(name)
			}
			i = j
			continue
		}
		b.WriteByte(src[i])
		i++
	}
	return b.String()
}
