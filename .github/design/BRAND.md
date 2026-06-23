# fur — Brand

The fur mark is a geometric monogram: paired chevrons (`>>` / `<<`) flanking a
bracketed `[ ]`, rendered on a solid black tile. It reads as "navigation between
documents" — the core of what fur does — while staying legible down to 16px.

## Palette

| Token | Hex | Usage |
|-------|------|-------|
| Ink | `#000000` | Background tile, primary fill |
| Paper | `#ffffff` | Chevrons, negative space |
| Signal Pink | `#fd578e` | Right bracket / accent |
| Signal Blue | `#52cbfd` | Left bracket / accent (≈89% opacity in mark) |

Pink and blue are the only chromatic accents; keep them paired and balanced.
Do not introduce additional hues. On light surfaces, the black tile provides its
own contrast — do not place the mark on a busy or mid-tone background.

## Assets

| File | Purpose |
|------|---------|
| `.github/design/assets/logo.svg` | Vector master (1254×1254 viewBox, Inkscape source) |
| `.github/design/assets/logo.png` | 1254×1254 raster master |
| `internal/web/static/favicon.ico` | Multi-size icon (16/32/48/64) for browser tabs |
| `internal/web/static/favicon-{16,32,64,256,512}.png` | PNG fallbacks |
| `internal/web/static/apple-touch-icon.png` | 180×180 iOS home-screen icon |

### Regenerating rasters

Favicons derive from the PNG master via PIL (Lanczos downscale):

```bash
python3 - <<'PY'
from PIL import Image
src = Image.open(".github/design/assets/logo.png").convert("RGBA")
out = "internal/web/static"
for s in (16, 32, 64, 256, 512):
    src.resize((s, s), Image.LANCZOS).save(f"{out}/favicon-{s}.png", optimize=True)
src.resize((256,256), Image.LANCZOS).save(f"{out}/favicon.ico", sizes=[(16,16),(32,32),(48,48),(64,64)])
src.resize((180,180), Image.LANCZOS).save(f"{out}/apple-touch-icon.png", optimize=True)
PY
```

## Clear space & minimum size

- Maintain clear space ≥ 1× the bracket width on all sides.
- Minimum render size: 16px (favicon). Below that the brackets lose definition.
- Never recolor, rotate, stretch, or add effects (shadows, gradients, outlines).
