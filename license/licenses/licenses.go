package licenses

var List map[string]string

func init() {
	List = map[string]string{
		"zlib":              ZlibLicense,
		"unlicense":         Unlicense,
		"mit":               MitLicense,
		"apache":            ApacheLicense,
		"bsd-2":             Bsd2Clause,
		"bsd-3":             Bsd3Clause,
		"bzip2":             BzipLicense,
		"isc":               ISCLicense,
		"gpl-2":             Gpl2License,
		"gpl-3":             Gpl3License,
		"lgpl-2":            LGPL2License,
		"wtfpl":             WTFPlLicense,
		"afreeo":            AfreeoLicense,
		"hackmii":           HackMiiLicense,
		"artistic":          ArtisticLicense,
		"cc0":               CC0License,
		"allpermissive":     AllPermissiveLicense,
		"json":              JsonLicense,
		"lha":               LhaLicense,
		"allrightsreserved": AllRightsReserved,
		"mmpl":              MinecraftMod,
		"nwhml":             TumblrLicense,
		"pftus":             PftusLicense,
		"bola":              BOLALicense,
		"downloadmii":       DownloadMiiLicense,
		"sqlite":            SQLiteBlessing,
		"fair":              FairLicense,
		"yolo":              YoloLicense,
	}
}

var YoloLicense = `                                  YOLO LICENSE
                             Version 1, July 10 2015

THIS SOFTWARE LICENSE IS PROVIDED "ALL CAPS" SO THAT YOU KNOW IT IS SUPER
SERIOUS AND YOU DON'T MESS AROUND WITH COPYRIGHT LAW BECAUSE YOU WILL GET IN
TROUBLE HERE ARE SOME OTHER BUZZWORDS COMMONLY IN THESE THINGS WARRANTIES
LIABILITY CONTRACT TORT LIABLE CLAIMS RESTRICTION MERCHANTABILITY SUBJECT TO
THE FOLLOWING CONDITIONS:

1. #yolo
2. #swag
3. #blazeit
`

var FairLicense = `Copyright {{.Year}} {{.Name}} <{{.Email}}>

Usage of the works is permitted provided that this instrument is retained
with the works, so that any entity that uses the works is notified of
this instrument.

DISCLAIMER: THE WORKS ARE WITHOUT WARRANTY.`

var SQLiteBlessing = `The author disclaims copyright to this source code.  In place of
a legal notice, here is a blessing:

    May you do good and not evil.
    May you find forgiveness for yourself and forgive others.
    May you share freely, never taking more than you give.`

var PftusLicense = `The P.F.T.U.S(Protected Free To Use Software) License
Copyright {{.Year}} {{.Name}} <{{.Email}}>, see the bottom of the document.
Version 1.1x

Unless required by applicable law or agreed to in writing, software
distributed under this License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
You are not allowed to re-license this Software, Product, Binary, Source
Code at all; unless you are the copyright holder.

Your use of this software indicates your acceptance of this license agreement
and warranty. We reserve rights to change this license or completely remove it
at ANY TIME without ANY notice.


Please notice that you are refereed to as "You", "Your" and "Mirrorer" and
"Re-distributor"

# SOURCE CODE
    1. You do not have permissions to use this code to make a profit in ANY
       POSSIBLE WAY, NOR are you allowed to use it for competing purposes[1];
       contributing to the source code is allowed provided you do it either
       via forking or similar way AND you don't make any kind of profit from it
       unless you were specifically hired to contribute.
    2. Mirroring is allowed as long as the mirrorer don't make any profit from
       mirroring of the code.

# BINARIES
    1. Binaries may not be re-distributed at all[2]
    2. Mirroring is allowed as long as the mirrorer don't make any profit
       from mirroring the produced binaries.

# DOCUMENTATION
    1. Redistribution of the included documentation is allowed as long as the
       re-distributor comply the the following term(s):
        a. You shall not make any more profit from it than what the upkeep of
           the documentation costs.
        b. You shall link back to the original documentation in the header of
           the redistribution web-page.
    2. Re-writing new documentation is allowed as long as it comply to the
       following term(s):
        a. It's clearly stated in the documentation that it isn't official.
        b. You are allowed to make a profit from it; as long as it is under
           10 000 USD annually.

Please notice that you can request written permission from the owner of this
Source Code/Binary or Project for Redistribution, using this software
for competing purposes or/and bypass the whole license.

[1] "competing purposes" means ANY software on ANY platform that is designed to
be used in similar fashion(eg serves the same or similar purposes as the
original software).

[2] This doesn't apply to the following domains:
    * github.com
    * AND any domain with written permissions

Copyright {{.Year}} {{.Name}} <{{.Email}}>, changing is not permitted,
redistribution is allowed. Some rights reserved.`

var BOLALicense = `I don't like licenses, because I don't like having to worry about all this
legal stuff just for a simple piece of software I don't really mind anyone
using. But I also believe that it's important that people share and give back;
so I'm placing this work under the following license.


BOLA - Buena Onda License Agreement (v1.1)
------------------------------------------

This work is provided 'as-is', without any express or implied warranty. In no
event will the authors be held liable for any damages arising from the use of
this work.

To all effects and purposes, this work is to be considered Public Domain.


However, if you want to be "buena onda", you should:

1. Not take credit for it, and give proper recognition to the authors.
2. Share your modifications, so everybody benefits from them.
3. Do something nice for the authors.
4. Help someone who needs it: sign up for some volunteer work or help your
   neighbour paint the house.
5. Don't waste. Anything, but specially energy that comes from natural
   non-renewable resources. Extra points if you discover or invent something
   to replace them.
6. Be tolerant. Everything that's good in nature comes from cooperation.`

var DownloadMiiLicense = `DownloadMii License
    Version 1.0x

Copyright (c) {{.Year}} {{.Name}} <{{.Email}}>
All rights reserved.

You do not have permissions to use this code to make a profit[1] in anyway,
nor to use it for competing purposes; contributing via forking and such is
allowed.

Mirroring is allowed as long as the mirrorer don't make any profit from
mirroring of code and produced binaries.

[1] If you however are developing software that is not competing with this
software you're allowed to make profit from it.`

var TumblrLicense = `Copyright (c) {{.Year}} {{.Name}} <{{.Email}}>

Non-White-Heterosexual-Male License

If you are not a white heterosexual male you are permitted to copy,
sell and use this work in any manner you choose without need to
include any attribution you do not see fit. You are asked as a
courtesy to retain this license in any derivatives but you are not
required. If you are a white heterosexual male you are provided the
same permissions (reuse, modification, resale) but are required to
include this license in any documentation and any public facing
derivative. You are also required to include attribution to the
original author or to an author responsible for redistribution
of a derivative.

For additional details see http://nonwhiteheterosexualmalelicense.org`

var AllRightsReserved = `Copyright (c) {{.Year}} {{.Name}} <{{.Email}}>

All rights reserved.`

var LhaLicense = `Copyright (c) {{.Year}} {{.Name}} <{{.Email}}>

Original Authors License Statement (from man/lha.man, translated by
Osamu Aoki <debian@aokiconsulting.com>):

   Permission is given for redistribution, copy, and modification provided
   following conditions are met.

   1. Do not remove copyright clause.
   2. Distribution shall conform:
    a. The content of redistribution (i.e., source code, documentation,
       and reference guide for programmers) shall include original contents.
       If contents are modified, the document clearly indicating
       the fact of modification must be included.
    b. If LHa is redistributed with added values, you must put your best
       effort to include them (Translator comment: If read literally,
       original Japanese was unclear what "them" means here.  But
       undoubtedly this "them" means source code for the added value
       portion and this is a typical Japanese sloppy writing style to
       abbreviate as such)  Also the document clearly indicating that
       added value was added must be included.
    c. Binary only distribution is not allowed (including added value
       ones.)
   3. You need to put effort to distribute the latest version (This is not
      your duty).

      NB: Distribution on Internet is free.  Please notify me by e-mail or
      other means prior to the distribution if distribution is done through
      non-Internet media (Magazine, CDROM etc.)  If not, make sure to Email
      me later.

   4. Any damage caused by the existence and use of this program will not
      be compensated.

   5. Author will not be responsible to correct errors even if program is
      defective.

   6. This program, either as a part of this or as a whole of this, may be
      included into other programs.  In this case, that program is not LHa
      and can not call itself LHa.

   7. For commercial use, in addition to above conditions, following
      condition needs to be met.

      a.  The program whose content is mainly this program can not be used
          commercially.
      b.  If the recipient of commercial use deems inappropriate as a
          program user, you must not distribute.
      c.  If used as a method for the installation, you must not force
          others to use this program.  In this case, commercial user will
          perform its work while taking full responsibility of its outcome.
      d.  If added value is done under the commercial use by using this
          program, commercial user shall provide its support.


(Osamu Aoki also comments:
   Here "commercial" may be interpreted as "for-fee".  "Added value" seems
   to mean "feature enhancement".  )

Translated License Statement by Tsugio Okamoto (translated by
GOTO Masanori <gotom@debian.org>):

   It's free to distribute on the network, but if you distribute for
   the people who cannot access the network (by magazine or CD-ROM),
   please send E-Mail (Inter-Net address) to the author before the
   distribution. That's well where this software is appeard.
   If you cannot do, you must send me the E-Mail later.`

var JsonLicense = `Copyright (c) {{.Year}} {{.Name}} <{{.Email}}>

Permission is hereby granted, free of charge, to any person obtaining
a copy of this software and associated documentation files (the
"Software"), to deal in the Software without restriction, including
without limitation the rights to use, copy, modify, merge, publish,
distribute, sublicense, and/or sell copies of the Software, and to
permit persons to whom the Software is furnished to do so, subject to
the following conditions:

The above copyright notice and this permission notice shall be
included in all copies or substantial portions of the Software.

The Software shall be used for Good, not Evil.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.`

var AllPermissiveLicense = `Copyright (c) {{.Year}} {{.Name}} <{{.Email}}>

Copying and distribution of this file, with or without modification,
are permitted in any medium without royalty provided the copyright
notice and this notice are preserved.  This file is offered as-is,
without any warranty.`

var ZlibLicense = `Copyright (c) {{.Year}} {{.Name}} <{{.Email}}>

This software is provided 'as-is', without any express or implied
warranty. In no event will the authors be held liable for any damages
arising from the use of this software.

Permission is granted to anyone to use this software for any purpose,
including commercial applications, and to alter it and redistribute it
freely, subject to the following restrictions:

1. The origin of this software must not be misrepresented; you must not
   claim that you wrote the original software. If you use this software
   in a product, an acknowledgement in the product documentation would be
   appreciated but is not required.

2. Altered source versions must be plainly marked as such, and must not be
   misrepresented as being the original software.

3. This notice may not be removed or altered from any source distribution.`

var Unlicense = `This is free and unencumbered software released into the public domain.

Anyone is free to copy, modify, publish, use, compile, sell, or
distribute this software, either in source code form or as a compiled
binary, for any purpose, commercial or non-commercial, and by any
means.

In jurisdictions that recognize copyright laws, the author or authors
of this software dedicate any and all copyright interest in the
software to the public domain. We make this dedication for the benefit
of the public at large and to the detriment of our heirs and
successors. We intend this dedication to be an overt act of
relinquishment in perpetuity of all present and future rights to this
software under copyright law.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
IN NO EVENT SHALL THE AUTHORS BE LIABLE FOR ANY CLAIM, DAMAGES OR
OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE,
ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
OTHER DEALINGS IN THE SOFTWARE.

For more information, please refer to <http://unlicense.org/>`

var MitLicense = `Copyright (c) {{.Year}} {{.Name}} <{{.Email}}>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.`

var ApacheLicense = `Copyright {{.Year}} {{.Name}} <{{.Email}}>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.`

var Bsd2Clause = `Copyright (c) {{.Year}} {{.Name}} <{{.Email}}>
All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

* Redistributions of source code must retain the above copyright notice, this
  list of conditions and the following disclaimer.

* Redistributions in binary form must reproduce the above copyright notice,
  this list of conditions and the following disclaimer in the documentation
  and/or other materials provided with the distribution.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.`

var Gpl2License = `Copyright (C) {{.Year}} {{.Name}} <{{.Email}}>

This program is free software; you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation; either version 2 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License along
with this program; if not, write to the Free Software Foundation, Inc.,
51 Franklin Street, Fifth Floor, Boston, MA 02110-1301 USA.`

var Gpl3License = `Copyright (C) {{.Year}} {{.Name}} <{{.Email}}>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.`

var WTFPlLicense = `DO WHAT THE FUCK YOU WANT TO PUBLIC LICENSE
                  Version 2, December 2004

Copyright (C) {{.Year}} {{.Name}} <{{.Email}}>

Everyone is permitted to copy and distribute verbatim or modified
copies of this license document, and changing it is allowed as long
as the name is changed.

           DO WHAT THE FUCK YOU WANT TO PUBLIC LICENSE
  TERMS AND CONDITIONS FOR COPYING, DISTRIBUTION AND MODIFICATION

  0. You just DO WHAT THE FUCK YOU WANT TO.`

var AfreeoLicense = `Copyright (C) {{.Year}} {{.Name}} <{{.Email}}>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published
by the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.`

var HackMiiLicense = `Copyright (C) {{.Year}} {{.Name}} <{{.Email}}>

you are not allowed to in anyway use this material (including but is not
limited to: source code, images, binaries, documents) for use in
competeting purpuse in ANY WAY. You are also not allowed to redistribute
NOR mirror the code & binarys in any way, link to us instead!`

var ArtisticLicense = `               The Artistic License 2.0

           Copyright (c) {{.Year}} {{.Name}} <{{.Email}}>

     Everyone is permitted to copy and distribute verbatim copies
      of this license document, but changing it is not allowed.

Preamble

This license establishes the terms under which a given free software
Package may be copied, modified, distributed, and/or redistributed.
The intent is that the Copyright Holder maintains some artistic
control over the development of that Package while still keeping the
Package available as open source and free software.

You are always permitted to make arrangements wholly outside of this
license directly with the Copyright Holder of a given Package.  If the
terms of this license do not permit the full use that you propose to
make of the Package, you should contact the Copyright Holder and seek
a different licensing arrangement.

Definitions

    "Copyright Holder" means the individual(s) or organization(s)
    named in the copyright notice for the entire Package.

    "Contributor" means any party that has contributed code or other
    material to the Package, in accordance with the Copyright Holder's
    procedures.

    "You" and "your" means any person who would like to copy,
    distribute, or modify the Package.

    "Package" means the collection of files distributed by the
    Copyright Holder, and derivatives of that collection and/or of
    those files. A given Package may consist of either the Standard
    Version, or a Modified Version.

    "Distribute" means providing a copy of the Package or making it
    accessible to anyone else, or in the case of a company or
    organization, to others outside of your company or organization.

    "Distributor Fee" means any fee that you charge for Distributing
    this Package or providing support for this Package to another
    party.  It does not mean licensing fees.

    "Standard Version" refers to the Package if it has not been
    modified, or has been modified only in ways explicitly requested
    by the Copyright Holder.

    "Modified Version" means the Package, if it has been changed, and
    such changes were not explicitly requested by the Copyright
    Holder.

    "Original License" means this Artistic License as Distributed with
    the Standard Version of the Package, in its current version or as
    it may be modified by The Perl Foundation in the future.

    "Source" form means the source code, documentation source, and
    configuration files for the Package.

    "Compiled" form means the compiled bytecode, object code, binary,
    or any other form resulting from mechanical transformation or
    translation of the Source form.


Permission for Use and Modification Without Distribution

(1)  You are permitted to use the Standard Version and create and use
Modified Versions for any purpose without restriction, provided that
you do not Distribute the Modified Version.


Permissions for Redistribution of the Standard Version

(2)  You may Distribute verbatim copies of the Source form of the
Standard Version of this Package in any medium without restriction,
either gratis or for a Distributor Fee, provided that you duplicate
all of the original copyright notices and associated disclaimers.  At
your discretion, such verbatim copies may or may not include a
Compiled form of the Package.

(3)  You may apply any bug fixes, portability changes, and other
modifications made available from the Copyright Holder.  The resulting
Package will still be considered the Standard Version, and as such
will be subject to the Original License.


Distribution of Modified Versions of the Package as Source

(4)  You may Distribute your Modified Version as Source (either gratis
or for a Distributor Fee, and with or without a Compiled form of the
Modified Version) provided that you clearly document how it differs
from the Standard Version, including, but not limited to, documenting
any non-standard features, executables, or modules, and provided that
you do at least ONE of the following:

    (a)  make the Modified Version available to the Copyright Holder
    of the Standard Version, under the Original License, so that the
    Copyright Holder may include your modifications in the Standard
    Version.

    (b)  ensure that installation of your Modified Version does not
    prevent the user installing or running the Standard Version. In
    addition, the Modified Version must bear a name that is different
    from the name of the Standard Version.

    (c)  allow anyone who receives a copy of the Modified Version to
    make the Source form of the Modified Version available to others
    under

    (i)  the Original License or

    (ii)  a license that permits the licensee to freely copy,
    modify and redistribute the Modified Version using the same
    licensing terms that apply to the copy that the licensee
    received, and requires that the Source form of the Modified
    Version, and of any works derived from it, be made freely
    available in that license fees are prohibited but Distributor
    Fees are allowed.


Distribution of Compiled Forms of the Standard Version
or Modified Versions without the Source

(5)  You may Distribute Compiled forms of the Standard Version without
the Source, provided that you include complete instructions on how to
get the Source of the Standard Version.  Such instructions must be
valid at the time of your distribution.  If these instructions, at any
time while you are carrying out such distribution, become invalid, you
must provide new instructions on demand or cease further distribution.
If you provide valid instructions or cease distribution within thirty
days after you become aware that the instructions are invalid, then
you do not forfeit any of your rights under this license.

(6)  You may Distribute a Modified Version in Compiled form without
the Source, provided that you comply with Section 4 with respect to
the Source of the Modified Version.


Aggregating or Linking the Package

(7)  You may aggregate the Package (either the Standard Version or
Modified Version) with other packages and Distribute the resulting
aggregation provided that you do not charge a licensing fee for the
Package.  Distributor Fees are permitted, and licensing fees for other
components in the aggregation are permitted. The terms of this license
apply to the use and Distribution of the Standard or Modified Versions
as included in the aggregation.

(8) You are permitted to link Modified and Standard Versions with
other works, to embed the Package in a larger work of your own, or to
build stand-alone binary or bytecode versions of applications that
include the Package, and Distribute the result without restriction,
provided the result does not expose a direct interface to the Package.


Items That are Not Considered Part of a Modified Version

(9) Works (including, but not limited to, modules and scripts) that
merely extend or make use of the Package, do not, by themselves, cause
the Package to be a Modified Version.  In addition, such works are not
considered parts of the Package itself, and are not subject to the
terms of this license.


General Provisions

(10)  Any use, modification, and distribution of the Standard or
Modified Versions is governed by this Artistic License. By using,
modifying or distributing the Package, you accept this license. Do not
use, modify, or distribute the Package, if you do not accept this
license.

(11)  If your Modified Version has been derived from a Modified
Version made by someone other than you, you are nevertheless required
to ensure that your Modified Version complies with the requirements of
this license.

(12)  This license does not grant you the right to use any trademark,
service mark, tradename, or logo of the Copyright Holder.

(13)  This license includes the non-exclusive, worldwide,
free-of-charge patent license to make, have made, use, offer to sell,
sell, import and otherwise transfer the Package with respect to any
patent claims licensable by the Copyright Holder that are necessarily
infringed by the Package. If you institute patent litigation
(including a cross-claim or counterclaim) against any party alleging
that the Package constitutes direct or contributory patent
infringement, then this Artistic License to you shall terminate on the
date that such litigation is filed.

(14)  Disclaimer of Warranty:
THE PACKAGE IS PROVIDED BY THE COPYRIGHT HOLDER AND CONTRIBUTORS "AS
IS' AND WITHOUT ANY EXPRESS OR IMPLIED WARRANTIES. THE IMPLIED
WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE, OR
NON-INFRINGEMENT ARE DISCLAIMED TO THE EXTENT PERMITTED BY YOUR LOCAL
LAW. UNLESS REQUIRED BY LAW, NO COPYRIGHT HOLDER OR CONTRIBUTOR WILL
BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, OR CONSEQUENTIAL
DAMAGES ARISING IN ANY WAY OUT OF THE USE OF THE PACKAGE, EVEN IF
ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.`

var ISCLicense = `Copyright (C) {{.Year}} {{.Name}} <{{.Email}}>

Permission to use, copy, modify, and/or distribute this software for any
purpose with or without fee is hereby granted, provided that the above
copyright notice and this permission notice appear in all copies.

THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.`

var Bsd3Clause = `Copyright (C) {{.Year}} {{.Name}} <{{.Email}}>

All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

* Redistributions of source code must retain the above copyright notice, this
  list of conditions and the following disclaimer.

* Redistributions in binary form must reproduce the above copyright notice,
  this list of conditions and the following disclaimer in the documentation
  and/or other materials provided with the distribution.

* Neither the name of [project] nor the names of its
  contributors may be used to endorse or promote products derived from
  this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.`

var LGPL2License = `Copyright (C) {{.Year}} {{.Name}} <{{.Email}}>

This library is free software; you can redistribute it and/or
modify it under the terms of the GNU Lesser General Public
License as published by the Free Software Foundation; either
version 2.1 of the License, or (at your option) any later version.

This library is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
Lesser General Public License for more details.

You should have received a copy of the GNU Lesser General Public
License along with this library; if not, write to the Free Software
Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301
USA`

var BzipLicense = `Copyright (C) {{.Year}} {{.Name}} <{{.Email}}>

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions
are met:

1. Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.

2. The origin of this software must not be misrepresented; you must
not claim that you wrote the original software. If you use this
software in a product, an acknowledgment in the product
documentation would be appreciated but is not required.

3. Altered source versions must be plainly marked as such, and must
not be misrepresented as being the original software.

4. The name of the author may not be used to endorse or promote
products derived from this software without specific prior written
permission.

THIS SOFTWARE IS PROVIDED BY THE AUTHOR AS IS AND ANY EXPRESS
OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
ARE DISCLAIMED. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR ANY
DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE
GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY,
WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING
NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.`

var CC0License = `Creative Commons Legal Code

CC0 1.0 Universal

    CREATIVE COMMONS CORPORATION IS NOT A LAW FIRM AND DOES NOT PROVIDE
    LEGAL SERVICES. DISTRIBUTION OF THIS DOCUMENT DOES NOT CREATE AN
    ATTORNEY-CLIENT RELATIONSHIP. CREATIVE COMMONS PROVIDES THIS
    INFORMATION ON AN "AS-IS" BASIS. CREATIVE COMMONS MAKES NO WARRANTIES
    REGARDING THE USE OF THIS DOCUMENT OR THE INFORMATION OR WORKS
    PROVIDED HEREUNDER, AND DISCLAIMS LIABILITY FOR DAMAGES RESULTING FROM
    THE USE OF THIS DOCUMENT OR THE INFORMATION OR WORKS PROVIDED
    HEREUNDER.

Statement of Purpose

The laws of most jurisdictions throughout the world automatically confer
exclusive Copyright and Related Rights (defined below) upon the creator
and subsequent owner(s) (each and all, an "owner") of an original work of
authorship and/or a database (each, a "Work").

Certain owners wish to permanently relinquish those rights to a Work for
the purpose of contributing to a commons of creative, cultural and
scientific works ("Commons") that the public can reliably and without fear
of later claims of infringement build upon, modify, incorporate in other
works, reuse and redistribute as freely as possible in any form whatsoever
and for any purposes, including without limitation commercial purposes.
These owners may contribute to the Commons to promote the ideal of a free
culture and the further production of creative, cultural and scientific
works, or to gain reputation or greater distribution for their Work in
part through the use and efforts of others.

For these and/or other purposes and motivations, and without any
expectation of additional consideration or compensation, the person
associating CC0 with a Work (the "Affirmer"), to the extent that he or she
is an owner of Copyright and Related Rights in the Work, voluntarily
elects to apply CC0 to the Work and publicly distribute the Work under its
terms, with knowledge of his or her Copyright and Related Rights in the
Work and the meaning and intended legal effect of CC0 on those rights.

1. Copyright and Related Rights. A Work made available under CC0 may be
protected by copyright and related or neighboring rights ("Copyright and
Related Rights"). Copyright and Related Rights include, but are not
limited to, the following:

  i. the right to reproduce, adapt, distribute, perform, display,
     communicate, and translate a Work;
 ii. moral rights retained by the original author(s) and/or performer(s);
iii. publicity and privacy rights pertaining to a person's image or
     likeness depicted in a Work;
 iv. rights protecting against unfair competition in regards to a Work,
     subject to the limitations in paragraph 4(a), below;
  v. rights protecting the extraction, dissemination, use and reuse of data
     in a Work;
 vi. database rights (such as those arising under Directive 96/9/EC of the
     European Parliament and of the Council of 11 March 1996 on the legal
     protection of databases, and under any national implementation
     thereof, including any amended or successor version of such
     directive); and
vii. other similar, equivalent or corresponding rights throughout the
     world based on applicable law or treaty, and any national
     implementations thereof.

2. Waiver. To the greatest extent permitted by, but not in contravention
of, applicable law, Affirmer hereby overtly, fully, permanently,
irrevocably and unconditionally waives, abandons, and surrenders all of
Affirmer's Copyright and Related Rights and associated claims and causes
of action, whether now known or unknown (including existing as well as
future claims and causes of action), in the Work (i) in all territories
worldwide, (ii) for the maximum duration provided by applicable law or
treaty (including future time extensions), (iii) in any current or future
medium and for any number of copies, and (iv) for any purpose whatsoever,
including without limitation commercial, advertising or promotional
purposes (the "Waiver"). Affirmer makes the Waiver for the benefit of each
member of the public at large and to the detriment of Affirmer's heirs and
successors, fully intending that such Waiver shall not be subject to
revocation, rescission, cancellation, termination, or any other legal or
equitable action to disrupt the quiet enjoyment of the Work by the public
as contemplated by Affirmer's express Statement of Purpose.

3. Public License Fallback. Should any part of the Waiver for any reason
be judged legally invalid or ineffective under applicable law, then the
Waiver shall be preserved to the maximum extent permitted taking into
account Affirmer's express Statement of Purpose. In addition, to the
extent the Waiver is so judged Affirmer hereby grants to each affected
person a royalty-free, non transferable, non sublicensable, non exclusive,
irrevocable and unconditional license to exercise Affirmer's Copyright and
Related Rights in the Work (i) in all territories worldwide, (ii) for the
maximum duration provided by applicable law or treaty (including future
time extensions), (iii) in any current or future medium and for any number
of copies, and (iv) for any purpose whatsoever, including without
limitation commercial, advertising or promotional purposes (the
"License"). The License shall be deemed effective as of the date CC0 was
applied by Affirmer to the Work. Should any part of the License for any
reason be judged legally invalid or ineffective under applicable law, such
partial invalidity or ineffectiveness shall not invalidate the remainder
of the License, and in such case Affirmer hereby affirms that he or she
will not (i) exercise any of his or her remaining Copyright and Related
Rights in the Work or (ii) assert any associated claims and causes of
action with respect to the Work, in either case contrary to Affirmer's
express Statement of Purpose.

4. Limitations and Disclaimers.

 a. No trademark or patent rights held by Affirmer are waived, abandoned,
    surrendered, licensed or otherwise affected by this document.
 b. Affirmer offers the Work as-is and makes no representations or
    warranties of any kind concerning the Work, express, implied,
    statutory or otherwise, including without limitation warranties of
    title, merchantability, fitness for a particular purpose, non
    infringement, or the absence of latent or other defects, accuracy, or
    the present or absence of errors, whether or not discoverable, all to
    the greatest extent permissible under applicable law.
 c. Affirmer disclaims responsibility for clearing rights of other persons
    that may apply to the Work or any use thereof, including without
    limitation any person's Copyright and Related Rights in the Work.
    Further, Affirmer disclaims responsibility for obtaining any necessary
    consents, permissions or other rights required for any use of the
    Work.
 d. Affirmer understands and acknowledges that Creative Commons is not a
    party to this document and has no duty or obligation with respect to
    this CC0 or use of the Work.`

var MinecraftMod = `Minecraft Mod Public License
============================

Version 1.0.1

0. Definitions
--------------

Minecraft: Denotes a copy of the Minecraft game licensed by Mojang AB

User: Anybody that interacts with the software in one of the following ways:
   - play
   - decompile
   - recompile or compile
   - modify
   - distribute

Mod: The mod code designated by the present license, in source form, binary
form, as obtained standalone, as part of a wider distribution or resulting from
the compilation of the original or modified sources.

Dependency: Code required for the mod to work properly. This includes
dependencies required to compile the code as well as any file or modification
that is explicitely or implicitely required for the mod to be working.

1. Scope
--------

The present license is granted to any user of the mod. As a prerequisite,
a user must own a legally acquired copy of Minecraft

2. Liability
------------

This mod is provided 'as is' with no warranties, implied or otherwise. The owner
of this mod takes no responsibility for any damages incurred from the use of
this mod. This mod alters fundamental parts of the Minecraft game, parts of
Minecraft may not work with this mod installed. All damages caused from the use
or misuse of this mad fall on the user.

3. Play rights
--------------

The user is allowed to install this mod on a client or a server and to play
without restriction.

4. Modification rights
----------------------

The user has the right to decompile the source code, look at either the
decompiled version or the original source code, and to modify it.

5. Derivation rights
--------------------

The user has the rights to derive code from this mod, that is to say to
write code that extends or instanciate the mod classes or interfaces, refer to
its objects, or calls its functions. This code is known as "derived" code, and
can be licensed under a license different from this mod.

6. Distribution of original or modified copy rights
---------------------------------------------------

Is subject to distribution rights this entire mod in its various forms. This
include:
   - original binary or source forms of this mod files
   - modified versions of these binaries or source files, as well as binaries
     resulting from source modifications
   - patch to its source or binary files
   - any copy of a portion of its binary source files

The user is allowed to redistribute this mod partially, in totality, or
included in a distribution.

When distributing binary files, the user must provide means to obtain its
entire set of sources or modified sources at no costs.

All distributions of this mod must remain licensed under the MMPL.

All dependencies that this mod have on other mods or classes must be licensed
under conditions comparable to this version of MMPL, with the exception of the
Minecraft code and the mod loading framework (e.g. ModLoader, ModLoaderMP or
Bukkit).

Modified version of binaries and sources, as well as files containing sections
copied from this mod, should be distributed under the terms of the present
license.`
