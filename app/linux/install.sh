#!/bin/sh
# Adds TILFacts to your apps menu. Run once:  ./install.sh
# To remove it later:  ./install.sh --remove
set -e
here=$(cd "$(dirname "$0")" && pwd)
dest="${XDG_DATA_HOME:-$HOME/.local/share}"
apps="$dest/applications"

if [ "$1" = "--remove" ]; then
  rm -rf "$dest/tilfacts" "$apps/tilfacts.desktop"
  echo "TILFacts removed from your apps menu."
  exit 0
fi

mkdir -p "$dest/tilfacts" "$apps"
cp "$here/TILFacts" "$here/tilfacts.png" "$dest/tilfacts/"
chmod +x "$dest/tilfacts/TILFacts"
cat > "$apps/tilfacts.desktop" <<DESKTOP
[Desktop Entry]
Type=Application
Name=TILFacts
Comment=Today I learned, condensed
Exec="$dest/tilfacts/TILFacts"
Icon=$dest/tilfacts/tilfacts.png
Terminal=false
Categories=Network;News;
DESKTOP
command -v update-desktop-database >/dev/null 2>&1 && update-desktop-database "$apps" >/dev/null 2>&1 || true
echo "Done. Open your apps menu and click TILFacts."
