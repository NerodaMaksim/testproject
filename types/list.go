package types

type Element struct {
	Key   string
	Value string

	next *Element
	prev *Element
}

func (e *Element) Next() *Element {
	return e.next
}

func (e *Element) Prev() *Element {
	return e.prev
}

type List struct {
	root Element
}

func (l *List) IsEmpty() bool {
	return l.root.next == nil
}

func (l *List) First() *Element {
	return l.root.next
}

func (l *List) Last() *Element {
	return l.root.prev
}

func (l *List) Push(key string, value string) *Element {
	e := &Element{Key: key, Value: value}

	if l.root.prev == nil {
		l.root.next = e
		l.root.prev = e

		return e
	}

	e.prev = l.root.prev
	l.root.prev.next = e
	l.root.prev = e

	return e
}

func (l *List) Remove(e *Element) {
	if e.prev == nil {
		l.root.next = e.next
	} else {
		e.prev.next = e.next
	}

	if e.next == nil {
		l.root.prev = e.prev
	} else {
		e.next.prev = e.prev
	}

	e.next = nil
	e.prev = nil
}
