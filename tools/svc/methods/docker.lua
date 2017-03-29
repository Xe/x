local json = require "json"
local sh = require "sh"

if os.getenv("DIE_ON_ERROR") == "yes" then
  sh { abort = true }
end

-- given the name, docker image name and environment
-- variables for this service, deploy it via Docker
-- running locally.
function deploy(name, imagename, vars, settings)
  args = { "run", "-d", "--name", name, "--label", "xe.svc.name="..name}

  for k,v in pairs(vars) do
    table.insert(args, "--env")
    table.insert(args, k .. "=" .. v)
  end

  table.insert(args, imagename)

  local cmd = sh.docker(unpack(args))
  cmd:ok()

  local ctrid = cmd:lines()()
  return ctrid
end

-- given a container name, return a table of information
-- about it
function inspect(name)
  local obj = sh.docker("inspect", ctrid):combinedOutput()
  local tbl, err = json.decode(obj)
  if err ~= nil then
    error(err, obj)
  end

  return tbl
end

-- kill a container by a given name
function kill(name)
  sh.docker("rm", "-f", name):ok()
end
