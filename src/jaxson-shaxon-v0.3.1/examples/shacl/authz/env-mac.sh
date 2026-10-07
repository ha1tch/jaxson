# Copyright (c) 2026 haitch <h@ual.li>
# Licensed under the GNU General Public License, version 3.
# https://www.gnu.org/licenses/gpl-3.0.html
#
# Source this on macOS to put the Homebrew toolchain that setup-mac.sh installed
# (a JDK, Maven, Go) on PATH and to set JAVA_HOME. bench.sh and setup-mac.sh do
# it for themselves; this file is for a shell where you run things by hand:
#
#     . ./env-mac.sh
#
# Harmless where there is no Homebrew: it then changes nothing.

_brew=
for _c in /opt/homebrew/bin/brew /usr/local/bin/brew "$(command -v brew 2>/dev/null)"; do
    if [ -n "$_c" ] && [ -x "$_c" ]; then _brew=$_c; break; fi
done

if [ -n "$_brew" ]; then
    _prefix=$("$_brew" --prefix)
    _jdk=$("$_brew" --prefix openjdk 2>/dev/null || true)
    if [ -n "$_jdk" ] && [ -d "$_jdk/libexec/openjdk.jdk/Contents/Home" ]; then
        JAVA_HOME="$_jdk/libexec/openjdk.jdk/Contents/Home"
        export JAVA_HOME
        PATH="$JAVA_HOME/bin:$PATH"
    fi
    PATH="$_prefix/bin:$PATH"
    export PATH
fi
unset _brew _c _prefix _jdk
