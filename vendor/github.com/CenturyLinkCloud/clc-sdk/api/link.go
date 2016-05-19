package api

type Link struct {
	Rel   string   `json:"rel,omitempty"`
	Href  string   `json:"href,omitempty"`
	ID    string   `json:"id,omitempty"`
	Name  string   `json:"name,omitempty"`
	Verbs []string `json:"verbs,omitempty"`
}

type Links []Link

func (l Links) GetID(rel string) (bool, string) {
	if ok, v := l.GetLink(rel); ok {
		return true, (*v).ID
	}
	return false, ""
}

func (l Links) GetLink(rel string) (bool, *Link) {
	for _, v := range l {
		if v.Rel == rel {
			return true, &v
		}
	}
	return false, nil
}

type Customfields struct {
	ID           string `json:"id,omitempty"`
	Name         string `json:"name,omitempty"`
	Value        string `json:"value,omitempty"`
	Displayvalue string `json:"displayValue,omitempty"`
}
