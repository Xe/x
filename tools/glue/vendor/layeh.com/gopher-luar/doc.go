// Package luar provides custom type reflection to gopher-lua.
//
// Notice
//
// This package is currently in development, and its behavior may change. This
// message will be removed once the package is considered stable.
//
// Basic types
//
// Go bool, number types, string types, and nil values are converted to the
// equivalent Lua type.
//
// Example:
//  New(L, "Hello World")        =  lua.LString("Hello World")
//  New(L, uint(834))            =  lua.LNumber(uint(834))
//  New(L, map[string]int(nil))  =  lua.LNil
//
// Channels
//
// Channels have the following methods defined:
//  receive():    Receives data from the channel. Returns nil plus false if the
//                channel is closed.
//  send(data):   Sends data to the channel.
//  close():      Closes the channel.
//
// Taking the length (#) of a channel returns how many unread items are in its
// buffer.
//
// Example:
//  ch := make(chan string)
//  L.SetGlobal("ch", New(L, ch))
//  ---
//  ch:receive()      -- equivalent to v, ok := ch
//  ch:send("hello")  -- equivalent to ch <- "hello"
//  ch:close()        -- equivalent to close(ch)
//
// Functions
//
// Functions can be converted and called from Lua. The function arguments and
// return values are automatically converted from and to Lua types,
// respectively (see exception below).
//
// Example:
//  fn := func(name string, age uint) string {
//    return fmt.Sprintf("Hello %s, age %d", name, age)
//  }
//  L.SetGlobal("fn", New(L, fn))
//  ---
//  print(fn("Tim", 5)) -- prints "Hello Tim, age 5"
//
// A function that has the signature func(*luar.LState) int can bypass the
// automatic argument and return value conversion (see luar.LState
// documentation for example).
//
// A special conversion case happens when function returns a lua.LValue slice.
// In that case, luar automatically unpacks the slice.
//
// Example:
//  fn := func() []lua.LValue {
//    return []lua.LValue{lua.LString("Hello"), lua.LNumber(2.5)}
//  }
//  L.SetGlobal("fn", New(L, fn))
//  ---
//  x, y = fn()
//  print(x) -- prints "Hello"
//  print(y) -- prints "2.5"
//
// Maps
//
// Maps can be accessed and modified like a normal Lua table. The map's length
// can also be queried using the # operator.
//
// Rather than using Lua's pairs function to create an map iterator, calling
// the value (e.g. map_variable()) returns an iterator for the map.
//
// Example:
//  places := map[string]string{
//    "NA": "North America",
//    "EU": "European Union",
//  }
//  L.SetGlobal("places", New(L, places))
//  ---
//  print(#places)       -- prints "2"
//  print(places.NA)     -- prints "North America"
//  print(places["EU"])  -- prints "European Union"
//  for k, v in places() do  -- prints all keys and values of places
//    print(k .. ": " .. v)
//  end
//
// Slices
//
// Like maps, slices be indexed, be modified, and have their length
// queried. Additionally, the following methods are defined for slices:
//  append(items...):   Appends the items to the slice. Returns a slice with
//                      the items appended.
//  capacity():         Returns the slice capacity.
//
// For consistency with other Lua code, slices use one-based indexing.
//
// Example:
//  letters := []string{"a", "e", "i"}
//  L.SetGlobal("letters", New(L, letters))
//  ---
//  letters = letters:append("o", "u")
//
// Like maps, calling a slice (e.g. slice()) returns an iterator over its
// values.
//
// Arrays
//
// Arrays can be indexed and have their length queried. Only pointers to
// arrays can their contents modified.
//
// Like slices and maps, calling an array (e.g. array()) returns an iterator
// over its values.
//
// Example:
//  var arr [2]string
//  L.SetGlobal("arr", New(L, &arr))
//  ---
//  arr[1] = "Hello"
//  arr[2] = "World"
//
// Structs
//
// Structs can have their fields accessed and modified and their methods
// called.
//
// Example:
//  type Person {
//    Name string
//  }
//  func (p Person) SayHello() {
//    fmt.Printf("Hello, %s\n", p.Name)
//  }
//
//  tim := Person{"Tim"}
//  L.SetGlobal("tim", New(L, tim))
//  ---
//  tim:SayHello() -- same as tim:sayHello()
//
// The name of a struct field is determined by its tag:
//  "":   the field is accessed by its name and its name with a lowercase
//        first letter
//  "-":  the field is not accessible
//  else: the field is accessed by that value
//
// Example:
//  type Person struct {
//    Name   string `luar:"name"`
//    Age    int
//    Hidden bool   `luar:"-"`
//  }
//  ---
//  Person.Name   -> "name"
//  Person.Age    -> "Age", "age"
//  Person.Hidden -> Not accessible
//
// Pointers
//
// Pointers can be dereferenced using the unary minus (-) operator.
//
// Example:
//  str := "hello"
//  L.SetGlobal("strptr", New(L, &str))
//  ---
//  print(-strptr) -- prints "hello"
//
// The pointed to value can changed using the pow (^) operator.
//
// Example:
//  str := "hello"
//  L.SetGlobal("strptr", New(L, &str))
//  ---
//  print(str^"world") -- prints "world", and str's value is now "world"
//
// Pointers to struct and array values are returned when accessed via a struct
// field, array index, or slice index.
//
// Type methods
//
// Any array, channel, map, slice, or struct type that has methods defined on
// it can be called from Lua.
//
// On maps with key strings, map elements are returned before type methods.
//
// Example:
//  type mySlice []string
//  func (s mySlice) Len() int {
//      return len(s)
//  }
//
//  var s mySlice = []string{"Hello", "world"}
//  L.SetGlobal("s", New(L, s))
//  ---
//  print(s:len()) -- prints "2"
//
// Lua to Go conversions
//
// The Lua types are automatically converted to match the output Go type, as
// described below:
//
//  Lua type    Go kind/type
//  -----------------------------------------------------
//  LBool       bool
//              string ("true" or "false")
//  LChannel    chan lua.LValue
//  LNumber     numeric value
//              string (strconv.Itoa)
//  LFunction   func
//  LNilType    chan, func, interface, map, ptr, slice, unsafe pointer
//  LState      *lua.LState
//  LString     string
//  LTable      slice
//              map
//              struct
//              *struct
//  LUserData   underlying lua.LUserData.Value type
//
// Example creating a Go slice from Lua:
//  type Group struct {
//      Names []string
//  }
//
//  g := new(Group)
//  L.SetGlobal("g", luar.New(L, g))
//  ---
//  g.Names = {"Tim", "Frank", "George"}
//
// New types
//
// Type constructors can be created using NewType. When called, it returns a
// new variable which is of the same type that was passed to NewType. Its
// behavior is dependent on the kind of value passed, as described below:
//
//  Kind      Constructor arguments          Return value
//  -----------------------------------------------------
//  Channel   Buffer size (opt)              Channel
//  Map       None                           Map
//  Slice     Length (opt), Capacity (opt)   Slice
//  Default   None                           Pointer to the newly allocated value
//
// Example:
//  type Person struct {
//    Name string
//  }
//  L.SetGlobal("Person", NewType(L, Person{}))
//  ---
//  p = Person()
//  p.Name = "John"
//  print("Hello, " .. p.Name)  // prints "Hello, John"
//
// Thread safety
//
// This package accesses and modifies the Lua state's registry. This happens
// when functions like New are called, and potentially when luar-created values
// are used. It is your responsibility to ensure that concurrent access of the
// state's registry does not happen.
package luar // import "layeh.com/gopher-luar"
