-- expects glue, $ go get -u github.com/Xe/tools/glue
local sh = require "sh"
sh { abort = true }

if os.getenv("CGO_ENABLED") ~= "0" then
  error("CGO_ENABLED must be set to 1")
end

print "building glue..."
sh.go("build"):print()
sh.upx("--ultra-brute", "glue"):print()
sh.box("box.rb"):print()

print "releasing to docker hub"
sh.docker("push", "xena/glue"):print()

print "moving glue binary to $GOPATH/bin"
sh.mv("glue", (os.getenv("GOPATH") .. "/bin/glue"))

print "build/release complete"
