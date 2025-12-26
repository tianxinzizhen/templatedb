package test

type Test struct {
	Id   int32  `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type TestV1 struct {
	Id   *IdScan `json:"id,omitempty"`
	Name string  `json:"name,omitempty"`
}

type IdScan struct {
	Id int32 `json:"id,omitempty"`
}

func (ts *IdScan) Scan(dest any) error {
	ts.Id = int32(dest.(int64))
	return nil
}

func (ts *IdScan) Val() IdScan {
	return *ts
}

func (ts IdScan) ValPtr() *IdScan {
	return &ts
}
