# lingo

Very basic Golang library for i18n. There are others that do the job, but this is my take on the problem.

## Features:

1. Storing messages in JSON files.
2. Support for nested declarations.
3. Detecting language based on Request headers.
4. Very simple to use.

## Usage:

1. Import Lingo into your project

   ```go
     import "github.com/kortem/lingo"
   ```

1. Create a dir to store translations, and write them in JSON files named [locale].json. For example:

   ```
     en_US.json
     sr_RS.json
     de.json
     ...
   ```

   You can write nested JSON too.

   ```json
   {
     "main.title": "CutleryPlus",
     "main.subtitle": "Knives that put cut in cutlery.",
     "menu": {
       "home": "Home",
       "products": {
         "self": "Products",
         "forks": "Forks",
         "knives": "Knives",
         "spoons": "Spoons"
       }
     }
   }
   ```

1. Initialize a Lingo like this:

   ```go
     l := lingo.New("default_locale", "path/to/translations/dir")
   ```

1. Get bundle for specific locale via either `string`:

   ```go
     t1 := l.TranslationsForLocale("en_US")
     t2 := l.TranslationsForLocale("de_DE")
   ```

   This way Lingo will return the bundle for specific locale, or default if given is not found.
   Alternatively (or primarily), you can get it with `*http.Request`:

   ```go
     t := l.TranslationsForRequest(req)
   ```

   This way Lingo finds best suited locale via `Accept-Language` header, or if there is no match, returns default.
   `Accept-Language` header is set by the browser, so basically it will serve the language the user has set to his browser.

1. Once you get T instance just fire away!

   ```go
     r1 := t1.Value("main.subtitle")
     // "Knives that put cut in cutlery."
     r1 := t2.Value("main.subtitle")
     // "Messer, die legte in Besteck geschnitten."
     r3 := t1.Value("menu.products.self")
     // "Products"
     r5 := t1.Value("error.404", req.URL.Path)
     // "Page index.html not found!"
   ```

## Contributions:

I regard this little library as feature-complete, but if you have an idea on how to improve it, feel free to create issues. Also, pull requests are welcome. Enjoy!
