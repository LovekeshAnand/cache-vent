"""Builds the app icon from assets/logo.png (needs Pillow).

Writes cmd/cachevent/rsrc_windows_amd64.syso (icon group 1, picked up by
go build) and prints the small PNG data URI used by the UI mark and favicon
(cmd/cachevent/ui/index.html).
Run from the repo root: python assets/make_icon.py
"""
import base64, io, struct
from PIL import Image, ImageDraw

rgba = Image.open("assets/logo.png").convert("RGBA")
# The logo has a transparent background; flatten it onto white.
src = Image.alpha_composite(Image.new("RGBA", rgba.size, "white"), rgba).convert("RGB")
# Crop to the drawing: everything that isn't near-white.
mask = src.convert("L").point(lambda v: 255 if v < 235 else 0)
logo = src.crop(mask.getbbox())

def tile(size):
    """The logo centered on a white rounded square."""
    big = size * 4
    im = Image.new("RGBA", (big, big), (0, 0, 0, 0))
    ImageDraw.Draw(im).rounded_rectangle((0, 0, big - 1, big - 1), radius=big // 5, fill="white")
    w = int(big * 0.86)
    h = round(logo.height * w / logo.width)
    im.paste(logo.resize((w, h), Image.LANCZOS), ((big - w) // 2, (big - h) // 2))
    return im.resize((size, size), Image.LANCZOS)

def png(im):
    b = io.BytesIO()
    im.save(b, "PNG", optimize=True)
    return b.getvalue()

# UI mark: the logo itself at 2x for a 22px-tall slot.
h = 44
mark = logo.resize((round(logo.width * h / logo.height), h), Image.LANCZOS)
print("data:image/png;base64," + base64.b64encode(png(mark)).decode())

# Windows resources: RT_ICON 1..n plus RT_GROUP_ICON 1, as a COFF object with a
# .rsrc section. Data-entry RVAs are relocated against the section symbol.
sizes = [16, 20, 24, 32, 40, 48, 64, 256]
icons = [png(tile(s)) for s in sizes]
group = struct.pack("<HHH", 0, 1, len(icons)) + b"".join(
    struct.pack("<BBBBHHIH", s % 256, s % 256, 0, 0, 1, 32, len(d), i + 1)
    for i, (s, d) in enumerate(zip(sizes, icons)))

RT_ICON, RT_GROUP_ICON, LANG = 3, 14, 0x409
leaves = [(RT_ICON, i + 1, d) for i, d in enumerate(icons)] + [(RT_GROUP_ICON, 1, group)]

def directory(n):
    return struct.pack("<IIHHHH", 0, 0, 0, 0, 0, n)

types = sorted({t for t, _, _ in leaves})
# Layout: root dir, type dirs, name dirs (one per leaf, with its language entry), data entries, data.
root_size = 16 + 8 * len(types)
type_sizes = {t: 16 + 8 * sum(1 for x in leaves if x[0] == t) for t in types}
off = root_size
type_off = {}
for t in types:
    type_off[t] = off
    off += type_sizes[t]
name_off = {}
for i in range(len(leaves)):
    name_off[i] = off
    off += 16 + 8
entry_off = {}
for i in range(len(leaves)):
    entry_off[i] = off
    off += 16
data_off = {}
for i, (_, _, d) in enumerate(leaves):
    off = (off + 7) & ~7
    data_off[i] = off
    off += len(d)

buf = bytearray(off)
buf[0:16] = directory(len(types))
for k, t in enumerate(types):
    struct.pack_into("<II", buf, 16 + 8 * k, t, 0x80000000 | type_off[t])
    ids = [(n, i) for i, (tt, n, _) in enumerate(leaves) if tt == t]
    buf[type_off[t]:type_off[t] + 16] = directory(len(ids))
    for j, (n, i) in enumerate(sorted(ids)):
        struct.pack_into("<II", buf, type_off[t] + 16 + 8 * j, n, 0x80000000 | name_off[i])
relocs = []
for i, (_, _, d) in enumerate(leaves):
    buf[name_off[i]:name_off[i] + 16] = directory(1)
    struct.pack_into("<II", buf, name_off[i] + 16, LANG, entry_off[i])
    struct.pack_into("<IIII", buf, entry_off[i], data_off[i], len(d), 0, 0)
    relocs.append(entry_off[i])
    buf[data_off[i]:data_off[i] + len(d)] = d

raw_ptr = 20 + 40
reloc_ptr = raw_ptr + len(buf)
sym_ptr = reloc_ptr + 10 * len(relocs)
IMAGE_REL_AMD64_ADDR32NB = 3
out = struct.pack("<HHIIIHH", 0x8664, 1, 0, sym_ptr, 1, 0, 0)
out += struct.pack("<8sIIIIIIHHI", b".rsrc", 0, 0, len(buf), raw_ptr, reloc_ptr, 0, len(relocs), 0, 0x40000040)
out += bytes(buf)
out += b"".join(struct.pack("<IIH", r, 0, IMAGE_REL_AMD64_ADDR32NB) for r in relocs)
out += struct.pack("<8sIhHBB", b".rsrc", 0, 1, 0, 3, 0)  # section symbol
out += struct.pack("<I", 4)  # empty string table
open("cmd/cachevent/rsrc_windows_amd64.syso", "wb").write(out)
print("icon sizes", sizes, "syso", len(out), "bytes, mark", len(png(mark)), "bytes")
