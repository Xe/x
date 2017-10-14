realpath
========

![Travis last test result on master](https://api.travis-ci.org/yookoala/realpath.svg?branch=master "Travis last test result on master")

This is a implementation of realpath() function in Go (golang).

If you provide it with a valid relative path / alias path, it will return you
with a string of its real absolute path in the system.

The original version is created by Taru Karttunen in golang-nuts group. You
may read [the original post about it](https://groups.google.com/forum/?fromgroups#!topic/golang-nuts/htns6YWMp7s).


Installation
------------

```
go get github.com/yookoala/realpath
```


Usage
-----

```go
import "github.com/yookoala/realpath"

func main() {
	myRealpath, err := realpath.Realpath("/some/path")
}
```


License
-------
This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Lesser General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Lesser General Public License for more details.

You should have received a copy of the GNU Lesser General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
