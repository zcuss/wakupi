#!/usr/bin/env python3
"""Generate the Wakupi app icon: a chat bubble + AI sparkle on a
WhatsApp-green -> violet diagonal gradient rounded square.

Renders at 4x supersampling for smooth anti-aliased edges, then downscales.
Outputs:
  - build/appicon.png        (1024x1024, used by Wails for all platforms)
  - build/windows/icon.ico   (multi-size ICO for Windows)
"""
from PIL import Image, ImageDraw
import math

SS = 4                      # supersample factor
SIZE = 1024
S = SIZE * SS               # working size

# Palette
GREEN = (0x00, 0xa8, 0x84)  # WhatsApp green  #00a884
VIOLET = (0x7c, 0x3a, 0xed)  # violet         #7c3aed
WHITE = (255, 255, 255)


def lerp(a, b, t):
    return tuple(int(round(a[i] + (b[i] - a[i]) * t)) for i in range(3))


def diagonal_gradient(size, c0, c1):
    """Top-left c0 -> bottom-right c1 diagonal gradient."""
    grad = Image.new("RGB", (size, size))
    px = grad.load()
    maxd = (size - 1) * 2
    for y in range(size):
        for x in range(size):
            t = (x + y) / maxd
            px[x, y] = lerp(c0, c1, t)
    return grad


def rounded_mask(size, radius):
    m = Image.new("L", (size, size), 0)
    d = ImageDraw.Draw(m)
    d.rounded_rectangle([0, 0, size - 1, size - 1], radius=radius, fill=255)
    return m


def main():
    # --- background: gradient clipped to a rounded square ---
    base = Image.new("RGBA", (S, S), (0, 0, 0, 0))
    grad = diagonal_gradient(S, GREEN, VIOLET).convert("RGBA")
    mask = rounded_mask(S, radius=int(S * 0.235))
    base.paste(grad, (0, 0), mask)

    draw = ImageDraw.Draw(base)

    # --- chat bubble (white, rounded, with a tail) ---
    # Bubble body
    bx0, by0 = int(S * 0.235), int(S * 0.215)
    bx1, by1 = int(S * 0.765), int(S * 0.660)
    br = int(S * 0.090)
    draw.rounded_rectangle([bx0, by0, bx1, by1], radius=br, fill=WHITE)

    # Tail (lower-left), drawn as a triangle merged into the body
    tail = [
        (int(S * 0.300), int(by1 - S * 0.010)),
        (int(S * 0.300), int(S * 0.790)),
        (int(S * 0.430), int(by1 - S * 0.010)),
    ]
    draw.polygon(tail, fill=WHITE)

    # --- three "message" dots inside the bubble (chat feel) ---
    cy = (by0 + by1) // 2
    dot_r = int(S * 0.038)
    gap = int(S * 0.140)
    cx = (bx0 + bx1) // 2
    for dx in (-gap, 0, gap):
        col = lerp(GREEN, VIOLET, 0.5 + (dx / (gap * 4)))
        draw.ellipse(
            [cx + dx - dot_r, cy - dot_r, cx + dx + dot_r, cy + dot_r],
            fill=col,
        )

    # --- AI sparkle (4-point star) top-right, overlapping bubble corner ---
    def sparkle(cx, cy, r_long, r_short, fill):
        pts = []
        for i in range(8):
            ang = math.pi / 2 - i * (math.pi / 4)
            r = r_long if i % 2 == 0 else r_short
            pts.append((cx + r * math.cos(ang), cy - r * math.sin(ang)))
        draw.polygon(pts, fill=fill)

    sx, sy = int(S * 0.775), int(S * 0.250)
    sparkle(sx, sy, int(S * 0.130), int(S * 0.045), WHITE)
    # small secondary sparkle
    sparkle(int(S * 0.690), int(S * 0.115), int(S * 0.055), int(S * 0.020), WHITE)

    # --- downscale for anti-aliasing ---
    icon = base.resize((SIZE, SIZE), Image.LANCZOS)
    icon.save("appicon.png")
    print("wrote appicon.png", icon.size)

    # --- Windows .ico (multi-size) ---
    ico_sizes = [256, 128, 64, 48, 32, 16]
    icon.save(
        "windows/icon.ico",
        format="ICO",
        sizes=[(s, s) for s in ico_sizes],
    )
    print("wrote windows/icon.ico", ico_sizes)


if __name__ == "__main__":
    main()
