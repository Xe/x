// Code generated by templ - DO NOT EDIT.

// templ: version: v0.2.731
package main

//lint:file-ignore SA4006 This context is only used if a nested component is present.

import (
	"github.com/a-h/templ"
	templruntime "github.com/a-h/templ/runtime"
)

func base(title string, body templ.Component) templ.Component {
	return templruntime.GeneratedTemplate(func(templ_7745c5c3_Input templruntime.GeneratedComponentInput) (templ_7745c5c3_Err error) {
		templ_7745c5c3_W, ctx := templ_7745c5c3_Input.Writer, templ_7745c5c3_Input.Context
		templ_7745c5c3_Buffer, templ_7745c5c3_IsBuffer := templruntime.GetBuffer(templ_7745c5c3_W)
		if !templ_7745c5c3_IsBuffer {
			defer func() {
				templ_7745c5c3_BufErr := templruntime.ReleaseBuffer(templ_7745c5c3_Buffer)
				if templ_7745c5c3_Err == nil {
					templ_7745c5c3_Err = templ_7745c5c3_BufErr
				}
			}()
		}
		ctx = templ.InitializeContext(ctx)
		templ_7745c5c3_Var1 := templ.GetChildren(ctx)
		if templ_7745c5c3_Var1 == nil {
			templ_7745c5c3_Var1 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<!doctype html><html lang=\"en\"><head><meta charset=\"UTF-8\"><meta http-equiv=\"X-UA-Compatible\" content=\"IE=edge\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\"><link rel=\"stylesheet\" href=\"https://cdn.xeiaso.net/static/pkg/iosevka/family.css\"><link rel=\"stylesheet\" href=\"/static/styles.css\"><title>")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		var templ_7745c5c3_Var2 string
		templ_7745c5c3_Var2, templ_7745c5c3_Err = templ.JoinStringErrs(title)
		if templ_7745c5c3_Err != nil {
			return templ.Error{Err: templ_7745c5c3_Err, FileName: `sanguisuga.templ`, Line: 12, Col: 17}
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var2))
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("</title><style>\n        [x-cloak] {\n            display: none\n        }\n    </style><script src=\"/static/alpine.js\" defer></script></head><body class=\"flex items-start justify-center h-full bg-gray-50\"><div class=\"w-full max-w-4xl px-2\"><nav class=\"my-4\"><a class=\"font-semibold underline text-gray-900 hover:text-gray-500\" href=\"/\">sanguisuga</a> - <a class=\"font-semibold underline text-gray-900 hover:text-gray-500\" href=\"/anime\">Anime</a> - <a href=\"/tv\" class=\"font-semibold underline text-gray-900 hover:text-gray-500\" href=\"/tv\">TV</a></nav><h1 class=\"mb-2 mt-0 text-3xl font-medium leading-tight text-primary\">")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		var templ_7745c5c3_Var3 string
		templ_7745c5c3_Var3, templ_7745c5c3_Err = templ.JoinStringErrs(title)
		if templ_7745c5c3_Err != nil {
			return templ.Error{Err: templ_7745c5c3_Err, FileName: `sanguisuga.templ`, Line: 33, Col: 12}
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var3))
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("</h1>")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		templ_7745c5c3_Err = body.Render(ctx, templ_7745c5c3_Buffer)
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<footer class=\"mt-2 bg-neutral-200 text-center dark:bg-neutral-700 lg:text-left\"><div class=\"p-4 text-neutral-700 dark:text-neutral-200\">From <a class=\"text-neutral-800 dark:text-neutral-400\" href=\"https://within.website\">Within</a>.</div></footer></div></body></html>")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		return templ_7745c5c3_Err
	})
}

func notFoundPage() templ.Component {
	return templruntime.GeneratedTemplate(func(templ_7745c5c3_Input templruntime.GeneratedComponentInput) (templ_7745c5c3_Err error) {
		templ_7745c5c3_W, ctx := templ_7745c5c3_Input.Writer, templ_7745c5c3_Input.Context
		templ_7745c5c3_Buffer, templ_7745c5c3_IsBuffer := templruntime.GetBuffer(templ_7745c5c3_W)
		if !templ_7745c5c3_IsBuffer {
			defer func() {
				templ_7745c5c3_BufErr := templruntime.ReleaseBuffer(templ_7745c5c3_Buffer)
				if templ_7745c5c3_Err == nil {
					templ_7745c5c3_Err = templ_7745c5c3_BufErr
				}
			}()
		}
		ctx = templ.InitializeContext(ctx)
		templ_7745c5c3_Var4 := templ.GetChildren(ctx)
		if templ_7745c5c3_Var4 == nil {
			templ_7745c5c3_Var4 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<p class=\"my-4\">If you expected to find a page here, this ain't it chief. Try <a href=\"/\">going home</a>.</p>")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		return templ_7745c5c3_Err
	})
}

func indexPage() templ.Component {
	return templruntime.GeneratedTemplate(func(templ_7745c5c3_Input templruntime.GeneratedComponentInput) (templ_7745c5c3_Err error) {
		templ_7745c5c3_W, ctx := templ_7745c5c3_Input.Writer, templ_7745c5c3_Input.Context
		templ_7745c5c3_Buffer, templ_7745c5c3_IsBuffer := templruntime.GetBuffer(templ_7745c5c3_W)
		if !templ_7745c5c3_IsBuffer {
			defer func() {
				templ_7745c5c3_BufErr := templruntime.ReleaseBuffer(templ_7745c5c3_Buffer)
				if templ_7745c5c3_Err == nil {
					templ_7745c5c3_Err = templ_7745c5c3_BufErr
				}
			}()
		}
		ctx = templ.InitializeContext(ctx)
		templ_7745c5c3_Var5 := templ.GetChildren(ctx)
		if templ_7745c5c3_Var5 == nil {
			templ_7745c5c3_Var5 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<p class=\"mt-4 max-w-xl\">Welcome to sanguisuga. This is a tool that will help you leech content from private trackers and XDCC bots. The following options are available:</p><ul class=\"list-disc list-inside mt-4\"><li><a class=\"font-semibold underline text-gray-900 hover:text-gray-500\" href=\"/anime\">Anime</a></li><li><a class=\"font-semibold underline text-gray-900 hover:text-gray-500\" href=\"/tv\">TV</a> (western TV)</li></ul><p class=\"my-4 max-w-xl\">Thank you for following the development of sanguisuga.</p>")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		return templ_7745c5c3_Err
	})
}

func animePage() templ.Component {
	return templruntime.GeneratedTemplate(func(templ_7745c5c3_Input templruntime.GeneratedComponentInput) (templ_7745c5c3_Err error) {
		templ_7745c5c3_W, ctx := templ_7745c5c3_Input.Writer, templ_7745c5c3_Input.Context
		templ_7745c5c3_Buffer, templ_7745c5c3_IsBuffer := templruntime.GetBuffer(templ_7745c5c3_W)
		if !templ_7745c5c3_IsBuffer {
			defer func() {
				templ_7745c5c3_BufErr := templruntime.ReleaseBuffer(templ_7745c5c3_Buffer)
				if templ_7745c5c3_Err == nil {
					templ_7745c5c3_Err = templ_7745c5c3_BufErr
				}
			}()
		}
		ctx = templ.InitializeContext(ctx)
		templ_7745c5c3_Var6 := templ.GetChildren(ctx)
		if templ_7745c5c3_Var6 == nil {
			templ_7745c5c3_Var6 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<h2 class=\"my-4 text-2xl font-medium leading-tight text-primary\">Shows</h2><div x-data=\"{ shows: [] }\" x-init=\"shows = await (await fetch(&#39;/api/anime/list&#39;)).json()\"><table class=\"border-collapse w-full border border-slate-400 bg-white text-sm shadow-sm\"><thead class=\"bg-slate-50\"><tr><th class=\"w-1/2 border border-slate-300 font-semibold p-4 text-slate-900 text-left\">Title</th><th class=\"w-1/2 border border-slate-300 font-semibold p-4 text-slate-900 text-left\">Disk Path</th><th class=\"w-1/2 border border-slate-300 font-semibold p-4 text-slate-900 text-left\"></th></tr></thead><template x-for=\"show in shows\"><tr><td class=\"border border-slate-300 p-4 text-slate-500\" x-text=\"show.title\"></td><td class=\"border border-slate-300 p-4 text-slate-500\"><code x-text=\"show.diskPath\"></code></td><td class=\"border border-slate-300 p-4 text-slate-500\"><button class=\"bg-gray-700 hover:bg-gray-800 disabled:opacity-50 text-white p-2\" @click=\"(e) =&gt; {\n                            fetch(&#39;/api/anime/untrack&#39;, {\n                                method: &#39;POST&#39;,\n                                body: JSON.stringify(show),\n                            })\n                                .then(() =&gt; e.target.parentElement.parentElement.remove())\n                                .catch((e) =&gt; alert(e))\n                            }\">🗑️</button></td></tr></template></table></div><h2 class=\"my-4 text-2xl font-medium leading-tight text-primary\">Downloads</h2><div x-data=\"{ open: false }\"><button class=\"bg-gray-700 hover:bg-gray-800 disabled:opacity-50 text-white p-2 mb-2\" @click=\"open = ! open\">Show</button><div class=\"bg-neutral-300 p-2\" x-show=\"open\" x-show=\"open\" x-transition:enter=\"transition ease-out duration-300\" x-transition:leave=\"transition ease-in duration-300\"><p class=\"mb-4\">")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		var templ_7745c5c3_Var7 string
		templ_7745c5c3_Var7, templ_7745c5c3_Err = templ.JoinStringErrs("Every one of these files is saved in the snatchlist and should be available on Plex. If a file is not available, you may need to check if someone put RAR files in a torrent. Again.")
		if templ_7745c5c3_Err != nil {
			return templ.Error{Err: templ_7745c5c3_Err, FileName: `sanguisuga.templ`, Line: 107, Col: 203}
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var7))
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("</p><div x-data=\"{ downloads: {} }\" x-init=\"downloads = await (await fetch(&#39;/api/anime/snatches&#39;)).json()\"><table class=\"border-collapse w-full border border-slate-400 bg-white text-sm shadow-sm\"><thead class=\"bg-slate-50\"><tr><th class=\"w-1/2 border border-slate-300 font-semibold p-4 text-slate-900 text-left\">Show</th><th class=\"w-1/2 border border-slate-300 font-semibold p-4 text-slate-900 text-left\">Episode</th><th class=\"w-1/2 border border-slate-300 font-semibold p-4 text-slate-900 text-left\">Method</th></tr></thead><template x-for=\"download in Object.values(downloads)\"><tr><td class=\"border border-slate-300 p-4 text-slate-500\" x-text=\"download.showName\"></td><td class=\"border border-slate-300 p-4 text-slate-500\" x-text=\"download.episode\"></td><td class=\"border border-slate-300 p-4 text-slate-500\">DCC</td></tr></template></table></div></div></div><h2 class=\"my-4 text-2xl font-medium leading-tight text-primary\">Track new show</h2><script>\n    function trackForm() {\n        return {\n            formData: {\n                title: \"\",\n                diskPath: \"/data/TV/\",\n                quality: \"1080p\",\n            },\n            message: '',\n            buttonLabel: \"Submit\",\n\n            submitData() {\n                this.message = ''\n\n                fetch('/api/anime/track', {\n                    method: 'POST',\n                    headers: { 'Content-Type': 'application/json' },\n                    body: JSON.stringify(this.formData),\n                })\n                    .then(() => {\n                        this.message = `Tracking ${this.formData.title}.`;\n                    })\n                    .catch((e) => {\n                        this.message = `error: ${e}`;\n                    });\n            }\n        }\n    }\n</script><div class=\"w-64 my-4\" x-data=\"trackForm()\"><div class=\"mb-4\"><label class=\"block mb-2\">Title:</label> <input type=\"text\" name=\"title\" class=\"border w-full p-1\" x-model=\"formData.title\"></div><div class=\"mb-4\"><label class=\"block mb-2\">Disk Path:</label> <input type=\"text\" name=\"diskPath\" class=\"border w-full p-1\" value=\"/data/TV/\" x-model=\"formData.diskPath\"></div><button class=\"bg-gray-700 hover:bg-gray-800 disabled:opacity-50 text-white w-full p-2\" x-text=\"buttonLabel\" @click=\"submitData\"></button><p x-text=\"message\"></p></div>")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		return templ_7745c5c3_Err
	})
}

func tvPage() templ.Component {
	return templruntime.GeneratedTemplate(func(templ_7745c5c3_Input templruntime.GeneratedComponentInput) (templ_7745c5c3_Err error) {
		templ_7745c5c3_W, ctx := templ_7745c5c3_Input.Writer, templ_7745c5c3_Input.Context
		templ_7745c5c3_Buffer, templ_7745c5c3_IsBuffer := templruntime.GetBuffer(templ_7745c5c3_W)
		if !templ_7745c5c3_IsBuffer {
			defer func() {
				templ_7745c5c3_BufErr := templruntime.ReleaseBuffer(templ_7745c5c3_Buffer)
				if templ_7745c5c3_Err == nil {
					templ_7745c5c3_Err = templ_7745c5c3_BufErr
				}
			}()
		}
		ctx = templ.InitializeContext(ctx)
		templ_7745c5c3_Var8 := templ.GetChildren(ctx)
		if templ_7745c5c3_Var8 == nil {
			templ_7745c5c3_Var8 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<h2 class=\"my-4 text-2xl font-medium leading-tight text-primary\">Shows</h2><div x-data=\"{ shows: [] }\" x-init=\"shows = await (await fetch(&#39;/api/tv/list&#39;)).json()\"><table class=\"border-collapse w-full border border-slate-400 bg-white text-sm shadow-sm\"><thead class=\"bg-slate-50\"><tr><th class=\"w-1/2 border border-slate-300 font-semibold p-4 text-slate-900 text-left\">Title</th><th class=\"w-1/2 border border-slate-300 font-semibold p-4 text-slate-900 text-left\">Disk Path</th><th class=\"w-1/2 border border-slate-300 font-semibold p-4 text-slate-900 text-left\"></th></tr></thead><template x-for=\"show in shows\"><tr><td class=\"border border-slate-300 p-4 text-slate-500\" x-text=\"show.title\"></td><td class=\"border border-slate-300 p-4 text-slate-500\"><code x-text=\"show.diskPath\"></code></td><td class=\"border border-slate-300 p-4 text-slate-500\"><button class=\"bg-gray-700 hover:bg-gray-800 disabled:opacity-50 text-white p-2\" @click=\"(e) =&gt; {\n                            fetch(&#39;/api/tv/untrack&#39;, {\n                                method: &#39;POST&#39;,\n                                body: JSON.stringify(show),\n                            })\n                                .then(() =&gt; e.target.parentElement.parentElement.remove())\n                                .catch((e) =&gt; alert(e))\n                            }\">🗑️</button></td></tr></template></table></div><h2 class=\"my-4 text-2xl font-medium leading-tight text-primary\">Downloads</h2><div x-data=\"{ open: false }\"><button class=\"bg-gray-700 hover:bg-gray-800 disabled:opacity-50 text-white p-2 mb-2\" @click=\"open = ! open\">Show</button><div class=\"bg-neutral-300 p-2\" x-show=\"open\" x-show=\"open\" x-transition:enter=\"transition ease-out duration-300\" x-transition:leave=\"transition ease-in duration-300\"><p class=\"mb-4\">")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		var templ_7745c5c3_Var9 string
		templ_7745c5c3_Var9, templ_7745c5c3_Err = templ.JoinStringErrs("Every one of these files is saved in the snatchlist and should be available on Plex. If a file is not available, you may need to check if someone put RAR files in a torrent. Again.")
		if templ_7745c5c3_Err != nil {
			return templ.Error{Err: templ_7745c5c3_Err, FileName: `sanguisuga.templ`, Line: 248, Col: 188}
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var9))
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("</p><div x-data=\"{ downloads: {} }\" x-init=\"downloads = await (await fetch(&#39;/api/tv/snatches&#39;)).json()\"><table class=\"border-collapse w-full border border-slate-400 bg-white text-sm shadow-sm\"><thead class=\"bg-slate-50\"><tr><th class=\"w-1/2 border border-slate-300 font-semibold p-4 text-slate-900 text-left\">Show + Episode</th><th class=\"w-1/2 border border-slate-300 font-semibold p-4 text-slate-900 text-left\">Torrent Link</th></tr></thead><template x-for=\"download in Object.keys(downloads)\"><tr><td class=\"border border-slate-300 p-4 text-slate-500 mx-auto\" x-text=\"download\"></td><td class=\"border border-slate-300 p-4 text-slate-500\" x-html=\"() =&gt; `&lt;a href=&#39;https://www.torrentleech.org/torrent/${downloads[download].TorrentID}&#39; target=&#39;_blank&#39;&gt;🔗&lt;/a&gt;`\"></td></tr></template></table></div></div></div><h2 class=\"my-4 text-2xl font-medium leading-tight text-primary\">Track new show</h2><script>\n  function trackForm() {\n    return {\n      formData: {\n        title: \"\",\n        diskPath: \"/data/TV/\",\n        quality: \"1080p\",\n      },\n      message: \"\",\n      buttonLabel: \"Submit\",\n\n      submitData() {\n        this.message = \"\";\n\n        fetch(\"/api/tv/track\", {\n          method: \"POST\",\n          headers: { \"Content-Type\": \"application/json\" },\n          body: JSON.stringify(this.formData),\n        })\n          .then(() => {\n            this.message = `Tracking ${this.formData.title}.`;\n          })\n          .catch((e) => {\n            this.message = `error: ${e}`;\n          });\n      },\n    };\n  }\n</script><div class=\"w-64 my-4\" x-data=\"trackForm()\"><div class=\"mb-4\"><label class=\"block mb-2\">Title:</label> <input type=\"text\" name=\"title\" class=\"border w-full p-1\" x-model=\"formData.title\"></div><div class=\"mb-4\"><label class=\"block mb-2\">Disk Path:</label> <input type=\"text\" name=\"diskPath\" class=\"border w-full p-1\" value=\"/data/TV/\" x-model=\"formData.diskPath\"></div><button class=\"bg-gray-700 hover:bg-gray-800 disabled:opacity-50 text-white w-full p-2\" x-text=\"buttonLabel\" @click=\"submitData\"></button><p x-text=\"message\"></p></div>")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		return templ_7745c5c3_Err
	})
}
