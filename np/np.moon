http = require "socket.http"
cjson = require "cjson"

main = do
  resp = http.request "http://ponyvillelive.com/api/nowplaying/index/station/ponyvillefm"

  obj = cjson.decode resp

  os.execute "status '" .. obj.result.streams[2].current_song.text .. "'"
