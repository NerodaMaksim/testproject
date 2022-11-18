package types

type Item struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Action string `json:"action"`
}

type ActionType string

// AddItem(), RemoveItem(), GetItem(), GetAllItems()
const (
	GetAllItems = "GetAllItems"
	GetItem     = "GetItem"
	AddItem     = "AddItem"
	RemoveItem  = "RemoveItem"
)
