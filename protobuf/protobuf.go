package protobuf // import "gitlab.com/ThatTomPerson/proteus/protobuf"

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"gitlab.com/ThatTomPerson/proteus/scanner"
)

// Package represents an unique .proto file with its own package definition.
type Package struct {
	Name     string
	Path     string
	Imports  []string
	Options  Options
	Messages []*Message
	Enums    []*Enum
	RPCs     []*RPC
}

// Import tries to import the given protobuf type to the current package.
// If the type requires no import at all, nothing will be done.
func (p *Package) Import(typ *ProtoType) {
	if typ.Import != "" && !p.isImported(typ.Import) {
		p.Imports = append(p.Imports, typ.Import)
	}
}

// ImportFromPath adds a new import from a Go path.
func (p *Package) ImportFromPath(path string) {
	file := filepath.Join(path, "generated.proto")
	if path != p.Path && !p.isImported(file) {
		p.Imports = append(p.Imports, filepath.Join(path, "generated.proto"))
	}
}

func (p *Package) isImported(file string) bool {
	for _, i := range p.Imports {
		if i == file {
			return true
		}
	}
	return false
}

// ServiceName returns the service name of the package.
func (p *Package) ServiceName() string {
	parts := strings.Split(p.Name, ".")
	last := parts[len(parts)-1]
	return strings.ToUpper(string(last[0])) + last[1:] + "Service"
}

// Message is the representation of a Protobuf message.
type Message struct {
	Docs     []string
	Name     string
	Reserved []uint
	Options  Options
	Fields   []*Field
}

// Reserve reserves a position in the message.
func (m *Message) Reserve(pos uint) {
	if !m.isReserved(pos) {
		m.Reserved = append(m.Reserved, pos)
	}
}

func (m *Message) isReserved(pos uint) bool {
	for _, r := range m.Reserved {
		if r == pos {
			return true
		}
	}
	return false
}

// Field is the representation of a protobuf message field.
type Field struct {
	Docs     []string
	Name     string
	Pos      int
	Repeated bool
	Type     Type
	Options  Options
}

// Options are the set of options given to a field, message or enum value.
type Options map[string]OptionValue

// Option name and value pair.
type Option struct {
	Name  string
	Value OptionValue
}

// Sorted returns a sorted set of options.
func (o Options) Sorted() []*Option {
	var names = make([]string, 0, len(o))
	for k := range o {
		names = append(names, k)
	}

	sort.Stable(sort.StringSlice(names))
	var opts = make([]*Option, len(o))
	for i, n := range names {
		opts[i] = &Option{Name: n, Value: o[n]}
	}

	return opts
}

// OptionValue is the common interface for the value of an option, which can be
// a literal value (a number, true, etc) or a string value ("foo").
type OptionValue interface {
	fmt.Stringer
	isOptionValue()
}

// LiteralValue is a literal option value like true, false or a number.
type LiteralValue struct {
	val string
}

// NewLiteralValue creates a new literal option value.
func NewLiteralValue(val string) LiteralValue {
	return LiteralValue{val}
}

func (LiteralValue) isOptionValue() {}
func (v LiteralValue) String() string {
	return v.val
}

// StringValue is a string option value.
type StringValue struct {
	val string
}

// NewStringValue creates a new string option value.
func NewStringValue(val string) StringValue {
	return StringValue{val}
}

func (StringValue) isOptionValue() {}
func (v StringValue) String() string {
	return fmt.Sprintf("%q", v.val)
}

// Type is the common interface of all possible types, which are named types,
// maps and basic types.
type Type interface {
	fmt.Stringer
	isType()
	SetSource(scanner.Type)
	Source() scanner.Type
	IsNullable() bool
}

// Named is a type which has a name and is defined somewhere else, maybe even
// in another package.
type Named struct {
	Package string
	Name    string
	// Generated reports whether the named type was generated by proteus or is
	// an user defined type.
	Generated bool
	Src       scanner.Type
}

// NewNamed creates a new Named type given its package and name.
func NewNamed(pkg, name string) *Named {
	return &Named{pkg, name, false, nil}
}

// NewGeneratedNamed creates a new Named type generated by proteus given its
// package and name.
func NewGeneratedNamed(pkg, name string) *Named {
	return &Named{pkg, name, true, nil}
}

func (n Named) String() string {
	return fmt.Sprintf("%s.%s", n.Package, n.Name)
}

// SetSource sets the scanner type source for a given protobuf named type.
func (n *Named) SetSource(t scanner.Type) {
	n.Src = t
}

// Source returns the scanner source type for a given protobuf type
func (n *Named) Source() scanner.Type {
	return n.Src
}

// IsNullable returns whether the type can be nulled or not.
func (n *Named) IsNullable() bool {
	if src := n.Source(); src != nil {
		return src.IsNullable()
	}

	return true // All named types in protobuf are nullable unless said otherwise.
}

// Alias represent a type declaration from one type to another.
type Alias struct {
	Type       Type
	Underlying Type
	Src        scanner.Type
}

// NewAlias returns a new Alias
func NewAlias(typ, underlying Type) *Alias {
	return &Alias{
		Type:       typ,
		Underlying: underlying,
	}
}

// SetSource sets the scanner type source for a given protobuf named type.
func (a *Alias) SetSource(t scanner.Type) {
	a.Src = t
}

// Source returns the scanner source type for a given protobuf type
func (a *Alias) Source() scanner.Type {
	return a.Src
}

func (a Alias) String() string {
	return a.Underlying.String()
}

// IsNullable returns whether an alias can be nulled or not
func (a Alias) IsNullable() bool {
	if src := a.Source(); src != nil {
		return src.IsNullable()
	}

	return a.Type.IsNullable()
}

// Basic is one of the basic types of protobuf.
type Basic struct {
	Name string
	Src  scanner.Type
}

// NewBasic creates a new basic type given its name.
func NewBasic(name string) *Basic {
	b := Basic{Name: name}
	return &b
}

func (b Basic) String() string {
	return b.Name
}

// SetSource sets the scanner type source for a given protobuf basic type.
func (b *Basic) SetSource(t scanner.Type) {
	b.Src = t
}

// Source returns the scanner source type for a given protobuf basic type
func (b *Basic) Source() scanner.Type {
	return b.Src
}

func (b *Basic) IsNullable() bool {
	return false
}

// Map is a key-value map type.
type Map struct {
	Key   Type
	Value Type
	Src   scanner.Type
}

// NewMap creates a new Map type with the key and value types given.
func NewMap(k, v Type) *Map {
	return &Map{k, v, nil}
}

func (m Map) String() string {
	return fmt.Sprintf("map<%s, %s>", m.Key, m.Value)
}

// SetSource sets the scanner type source for a given protobuf map type.
func (m *Map) SetSource(t scanner.Type) {
	m.Src = t
}

// Source returns the scanner source type for a given protobuf map type
func (m *Map) Source() scanner.Type {
	return m.Src
}

func (m *Map) IsNullable() bool {
	return m.Value.IsNullable()
}

func (*Named) isType() {}
func (*Basic) isType() {}
func (*Map) isType()   {}
func (*Alias) isType() {}

// Enum is the representation of a protobuf enumeration.
type Enum struct {
	Docs    []string
	Name    string
	Options Options
	Values  []*EnumValue
}

// EnumValue is a single value in an enumeration.
type EnumValue struct {
	Docs    []string
	Name    string
	Value   uint
	Options Options
}

// RPC is a single exposed RPC method in the RPC service.
type RPC struct {
	Docs []string
	Name string
	// Recv is the name of the receiver Go type. Empty if it's not a method.
	Recv string
	// Method is the name of the Go method or function.
	Method string
	// HasCtx reports whether the Go function accepts context.
	HasCtx bool
	// HasError reports whether the Go function returns an error.
	HasError bool
	// IsVariadic reports whether the Go function is variadic or not.
	IsVariadic bool
	Input      Type
	Output     Type
	Options    Options
}
