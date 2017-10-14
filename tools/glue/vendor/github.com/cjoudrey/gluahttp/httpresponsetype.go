package gluahttp

import "github.com/yuin/gopher-lua"
import "net/http"

const luaHttpResponseTypeName = "http.response"

type luaHttpResponse struct {
	res      *http.Response
	body     lua.LString
	bodySize int
}

func registerHttpResponseType(module *lua.LTable, L *lua.LState) {
	mt := L.NewTypeMetatable(luaHttpResponseTypeName)
	L.SetField(mt, "__index", L.NewFunction(httpResponseIndex))

	L.SetField(module, "response", mt)
}

func newHttpResponse(res *http.Response, body *[]byte, bodySize int, L *lua.LState) *lua.LUserData {
	ud := L.NewUserData()
	ud.Value = &luaHttpResponse{
		res:      res,
		body:     lua.LString(*body),
		bodySize: bodySize,
	}
	L.SetMetatable(ud, L.GetTypeMetatable(luaHttpResponseTypeName))
	return ud
}

func checkHttpResponse(L *lua.LState) *luaHttpResponse {
	ud := L.CheckUserData(1)
	if v, ok := ud.Value.(*luaHttpResponse); ok {
		return v
	}
	L.ArgError(1, "http.response expected")
	return nil
}

func httpResponseIndex(L *lua.LState) int {
	res := checkHttpResponse(L)

	switch L.CheckString(2) {
	case "headers":
		return httpResponseHeaders(res, L)
	case "cookies":
		return httpResponseCookies(res, L)
	case "status_code":
		return httpResponseStatusCode(res, L)
	case "url":
		return httpResponseUrl(res, L)
	case "body":
		return httpResponseBody(res, L)
	case "body_size":
		return httpResponseBodySize(res, L)
	}

	return 0
}

func httpResponseHeaders(res *luaHttpResponse, L *lua.LState) int {
	headers := L.NewTable()
	for key, _ := range res.res.Header {
		headers.RawSetString(key, lua.LString(res.res.Header.Get(key)))
	}
	L.Push(headers)
	return 1
}

func httpResponseCookies(res *luaHttpResponse, L *lua.LState) int {
	cookies := L.NewTable()
	for _, cookie := range res.res.Cookies() {
		cookies.RawSetString(cookie.Name, lua.LString(cookie.Value))
	}
	L.Push(cookies)
	return 1
}

func httpResponseStatusCode(res *luaHttpResponse, L *lua.LState) int {
	L.Push(lua.LNumber(res.res.StatusCode))
	return 1
}

func httpResponseUrl(res *luaHttpResponse, L *lua.LState) int {
	L.Push(lua.LString(res.res.Request.URL.String()))
	return 1
}

func httpResponseBody(res *luaHttpResponse, L *lua.LState) int {
	L.Push(res.body)
	return 1
}

func httpResponseBodySize(res *luaHttpResponse, L *lua.LState) int {
	L.Push(lua.LNumber(res.bodySize))
	return 1
}
