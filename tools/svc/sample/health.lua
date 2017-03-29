-- This file will be run with glue: https://github.com/Xe/tools/tree/master/glue

local http = require "http"
local url = arg[1]

local resp, err = http.get(url, {
  headers = {
    ["X-SVC-Healthcheck"] = tostring(os.date())
  }
})

if err ~= nil then
  error(err)
end

print(resp)
