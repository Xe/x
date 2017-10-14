local re = require("re")
local a, b, i, s, t

-- quote
assert(re.quote("^$.?a") == [[\^\$\.\?a]])

-- find
function f(s, p)
  local i,e = string.find(s, p)
  if i then return string.sub(s, i, e) end
end
a,b = re.find('','')
assert(a == 1 and b == 0)
a, b = re.find("abcd efgh ijk", "cd e", 1, true)
assert(a == 3 and b == 6)
a, b = re.find("abcd efgh ijk", "cd e", -1, true)
assert(a == nil)
assert(not pcall(re.find, "aaa", "(aaaa"))
assert(re.find("aaaa", "(b|c)*") == nil)
a, b, s = re.find("abcd efgh ijk", "i([jk])")
assert(a == 11 and b == 12 and s == "j")

-- gsub
assert(not pcall(re.gsub, "aaa", "(aaaa", "${1}"))
s, a = re.gsub("aaaa", "bbbb", "")
assert(s == "aaaa" and a == 0)
assert(re.gsub("hello world", [[(\w+)]], "${1} ${1}") == "hello hello world world")
t = {name="lua", version="5.1"}
assert(re.gsub("$name-$version.tar.gz", [[\$(\w+)]], t) == "lua-5.1.tar.gz")
assert(re.gsub("name version", [[\w+]], t) == "lua 5.1")
assert(re.gsub("4+5 = $return 4+5$", "\\$(.*)\\$", function (s)
           return loadstring(s)()
         end) == "4+5 = 9")
assert(re.gsub("$ world", "\\w+", string.upper) == "$ WORLD")

-- gmatch
assert(not pcall(re.gmatch, "hello world", "(aaaaa"))
i = 1
for w in re.gmatch("hello world", "\\w+") do
  if i == 1 then
    assert(w == "hello")
  elseif i == 2 then
    assert(w == "world")
  end
  i = i + 1
end
assert(i == 3)

i = 1
for k, v in re.gmatch("from=world, to=Lua", "(\\w+)=(\\w+)") do
  if i == 1 then
    assert(k == "from" and v =="world")
  elseif i == 2 then
    assert(k == "to" and v =="Lua")
  end
  i = i + 1
end
assert(i == 3)

-- match
assert(not pcall(re.match, "hello world", "(aaaaa"))
assert(re.match("$$$ hello", "z") == nil)
assert(re.match("$$$ hello", "\\w+") == "hello")
assert(re.match("hello world", "\\w+", 6) == "world")
assert(re.match("hello world", "\\w+", -5) == "world")
a, b = re.match("from=world", "(\\w+)=(\\w+)")
assert(a == "from" and b == "world")
