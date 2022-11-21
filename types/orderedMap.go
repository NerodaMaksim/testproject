package types

type OrderedMap struct {
	dict map[string]*Element
	list List
}

func NewOrderedMap() *OrderedMap {
	return &OrderedMap{
		dict: make(map[string]*Element),
	}
}

func (m *OrderedMap) Set(key string, value string) bool {
	_, alreadyExist := m.dict[key]
	if alreadyExist {
		m.dict[key].Value = value
		return false
	}

	element := m.list.Push(key, value)
	m.dict[key] = element
	return true
}

func (m *OrderedMap) Get(key string) (value string, ok bool) {
	v, ok := m.dict[key]
	if ok {
		value = v.Value
	}

	return
}

func (m *OrderedMap) Delete(key string) (didDelete bool) {
	element, ok := m.dict[key]
	if ok {
		m.list.Remove(element)
		delete(m.dict, key)
	}

	return ok
}

func (m *OrderedMap) Keys() (keys []string) {
	keys = make([]string, 0, len(m.dict))
	for el := m.list.First(); el != nil; el = el.Next() {
		keys = append(keys, el.Key)
	}

	return
}
