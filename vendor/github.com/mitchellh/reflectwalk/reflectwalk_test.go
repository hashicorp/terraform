package reflectwalk

import (
	"reflect"
	"testing"
)

type TestEnterExitWalker struct {
	Locs []Location
}

func (t *TestEnterExitWalker) Enter(l Location) error {
	if t.Locs == nil {
		t.Locs = make([]Location, 0, 5)
	}

	t.Locs = append(t.Locs, l)
	return nil
}

func (t *TestEnterExitWalker) Exit(l Location) error {
	t.Locs = append(t.Locs, l)
	return nil
}

type TestPointerWalker struct {
	Ps []bool
}

func (t *TestPointerWalker) PointerEnter(v bool) error {
	t.Ps = append(t.Ps, v)
	return nil
}

func (t *TestPointerWalker) PointerExit(v bool) error {
	return nil
}

type TestPrimitiveWalker struct {
	Value reflect.Value
}

func (t *TestPrimitiveWalker) Primitive(v reflect.Value) error {
	t.Value = v
	return nil
}

type TestPrimitiveCountWalker struct {
	Count int
}

func (t *TestPrimitiveCountWalker) Primitive(v reflect.Value) error {
	t.Count += 1
	return nil
}

type TestPrimitiveReplaceWalker struct {
	Value reflect.Value
}

func (t *TestPrimitiveReplaceWalker) Primitive(v reflect.Value) error {
	v.Set(reflect.ValueOf("bar"))
	return nil
}

type TestMapWalker struct {
	MapVal reflect.Value
	Keys   []string
	Values []string
}

func (t *TestMapWalker) Map(m reflect.Value) error {
	t.MapVal = m
	return nil
}

func (t *TestMapWalker) MapElem(m, k, v reflect.Value) error {
	if t.Keys == nil {
		t.Keys = make([]string, 0, 1)
		t.Values = make([]string, 0, 1)
	}

	t.Keys = append(t.Keys, k.Interface().(string))
	t.Values = append(t.Values, v.Interface().(string))
	return nil
}

type TestSliceWalker struct {
	Count    int
	SliceVal reflect.Value
}

func (t *TestSliceWalker) Slice(v reflect.Value) error {
	t.SliceVal = v
	return nil
}

func (t *TestSliceWalker) SliceElem(int, reflect.Value) error {
	t.Count++
	return nil
}

type TestStructWalker struct {
	Fields []string
}

func (t *TestStructWalker) Struct(v reflect.Value) error {
	return nil
}

func (t *TestStructWalker) StructField(sf reflect.StructField, v reflect.Value) error {
	if t.Fields == nil {
		t.Fields = make([]string, 0, 1)
	}

	t.Fields = append(t.Fields, sf.Name)
	return nil
}

func TestTestStructs(t *testing.T) {
	var raw interface{}
	raw = new(TestEnterExitWalker)
	if _, ok := raw.(EnterExitWalker); !ok {
		t.Fatal("EnterExitWalker is bad")
	}

	raw = new(TestPrimitiveWalker)
	if _, ok := raw.(PrimitiveWalker); !ok {
		t.Fatal("PrimitiveWalker is bad")
	}

	raw = new(TestMapWalker)
	if _, ok := raw.(MapWalker); !ok {
		t.Fatal("MapWalker is bad")
	}

	raw = new(TestSliceWalker)
	if _, ok := raw.(SliceWalker); !ok {
		t.Fatal("SliceWalker is bad")
	}

	raw = new(TestStructWalker)
	if _, ok := raw.(StructWalker); !ok {
		t.Fatal("StructWalker is bad")
	}
}

func TestWalk_Basic(t *testing.T) {
	w := new(TestPrimitiveWalker)

	type S struct {
		Foo string
	}

	data := &S{
		Foo: "foo",
	}

	err := Walk(data, w)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if w.Value.Kind() != reflect.String {
		t.Fatalf("bad: %#v", w.Value)
	}
}

func TestWalk_Basic_Replace(t *testing.T) {
	w := new(TestPrimitiveReplaceWalker)

	type S struct {
		Foo string
		Bar []interface{}
	}

	data := &S{
		Foo: "foo",
		Bar: []interface{}{[]string{"what"}},
	}

	err := Walk(data, w)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if data.Foo != "bar" {
		t.Fatalf("bad: %#v", data.Foo)
	}
	if data.Bar[0].([]string)[0] != "bar" {
		t.Fatalf("bad: %#v", data.Bar)
	}
}

func TestWalk_EnterExit(t *testing.T) {
	w := new(TestEnterExitWalker)

	type S struct {
		A string
		M map[string]string
	}

	data := &S{
		A: "foo",
		M: map[string]string{
			"a": "b",
		},
	}

	err := Walk(data, w)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := []Location{
		WalkLoc,
		Struct,
		StructField,
		StructField,
		StructField,
		Map,
		MapKey,
		MapKey,
		MapValue,
		MapValue,
		Map,
		StructField,
		Struct,
		WalkLoc,
	}
	if !reflect.DeepEqual(w.Locs, expected) {
		t.Fatalf("Bad: %#v", w.Locs)
	}
}

func TestWalk_Interface(t *testing.T) {
	w := new(TestPrimitiveCountWalker)

	type S struct {
		Foo string
		Bar []interface{}
	}

	var data interface{} = &S{
		Foo: "foo",
		Bar: []interface{}{[]string{"bar", "what"}, "baz"},
	}

	err := Walk(data, w)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if w.Count != 4 {
		t.Fatalf("bad: %#v", w.Count)
	}
}

func TestWalk_Interface_nil(t *testing.T) {
	w := new(TestPrimitiveCountWalker)

	type S struct {
		Bar interface{}
	}

	var data interface{} = &S{}

	err := Walk(data, w)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestWalk_Map(t *testing.T) {
	w := new(TestMapWalker)

	type S struct {
		Foo map[string]string
	}

	data := &S{
		Foo: map[string]string{
			"foo": "foov",
			"bar": "barv",
		},
	}

	err := Walk(data, w)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(w.MapVal.Interface(), data.Foo) {
		t.Fatalf("Bad: %#v", w.MapVal.Interface())
	}

	expectedK := []string{"foo", "bar"}
	if !reflect.DeepEqual(w.Keys, expectedK) {
		t.Fatalf("Bad keys: %#v", w.Keys)
	}

	expectedV := []string{"foov", "barv"}
	if !reflect.DeepEqual(w.Values, expectedV) {
		t.Fatalf("Bad values: %#v", w.Values)
	}
}

func TestWalk_Pointer(t *testing.T) {
	w := new(TestPointerWalker)

	type S struct {
		Foo string
	}

	data := &S{
		Foo: "foo",
	}

	err := Walk(data, w)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := []bool{true, false}
	if !reflect.DeepEqual(w.Ps, expected) {
		t.Fatalf("bad: %#v", w.Ps)
	}
}

func TestWalk_Slice(t *testing.T) {
	w := new(TestSliceWalker)

	type S struct {
		Foo []string
	}

	data := &S{
		Foo: []string{"a", "b", "c"},
	}

	err := Walk(data, w)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(w.SliceVal.Interface(), data.Foo) {
		t.Fatalf("bad: %#v", w.SliceVal.Interface())
	}

	if w.Count != 3 {
		t.Fatalf("Bad count: %d", w.Count)
	}
}

func TestWalk_Struct(t *testing.T) {
	w := new(TestStructWalker)

	type S struct {
		Foo string
		Bar string
	}

	data := &S{
		Foo: "foo",
		Bar: "bar",
	}

	err := Walk(data, w)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := []string{"Foo", "Bar"}
	if !reflect.DeepEqual(w.Fields, expected) {
		t.Fatalf("bad: %#v", w.Fields)
	}
}
